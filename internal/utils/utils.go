package utils

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ExistsAndIsDirectory checks if the given path exists and is present in the directory.
func ExistsAndIsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)

	// Case 1: Path does not exist.
	if os.IsNotExist(err) {
		return false, err
	}

	// Case 2: Error Handling
	if err != nil {
		return false, fmt.Errorf("stat check failed for %s: %w", path, err)
	}

	// Case 3: Path Exists
	return info.IsDir(), nil
}

/*
WalkDir(root, fn) walks through all files and directories under root.
It calls fn(path, d, err) for each entry, including root itself.
The callback can handle each item or skip directories (e.g. return filepath.SkipDir).
Returns an error if the walk fails.
*/
func WalkAndAddFiles(root string, handleFile func(string) error) error {
	return filepath.WalkDir(root, func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %w", entryPath, err)
		}

		// Skip anything starting with `.`
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip remaining directories
		if d.IsDir() {
			return nil
		}

		// Handle file, but continue on error
		if err := handleFile(entryPath); err != nil {
			log.Printf("error handling file %s: %v", entryPath, err)
			return nil
		}
		return nil
	})
}

// StoreObject handles creating directories and writing compressed blob
func StoreObject(hashStr string, data []byte) error {
	dir := filepath.Join(".purr", "objects", hashStr[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	objectPath := filepath.Join(dir, hashStr[2:])
	return os.WriteFile(objectPath, data, 0644)
}

/*
*
PopulateAllIndexField creates an IndexEntry from the provided os.FileInfo and relative path.
It extracts file metadata, handling Windows-specific fields, and populates all index fields.
*/
func PopulateAllIndexField(fileInfo os.FileInfo, relPath string) IndexEntry {
	stat := fileInfo.Sys().(*syscall.Win32FileAttributeData)
	return IndexEntry{
		Ctime: time.Unix(0, stat.CreationTime.Nanoseconds()),
		Mtime: fileInfo.ModTime(),
		Dev:   0, // Not applicable on Windows
		Ino:   0, // Not applicable on Windows
		Mode:  uint32(fileInfo.Mode()),
		Uid:   0, // Not applicable on Windows
		Gid:   0, // Not applicable on Windows
		Size:  uint32(fileInfo.Size()),
		Stage: 0,
		Path:  relPath,
	}
}

// GetHEADCommit reads the current HEAD commit hash from the .purr directory.
// It handles both symbolic references (e.g., "ref: refs/heads/main") and detached HEAD states (direct commit hash).
// Returns the commit hash as a string, or an error if reading fails.
func GetHEADCommit() (string, error) {
	headPath := filepath.Join(".purr", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	ref := strings.TrimSpace(string(content)) //to clean \n ( Empty spaces)

	// Case1: HEAD --> ref: refs/heads/main or similar
	if strings.HasPrefix(ref, "ref:") {
		branchRef := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		branchPath := filepath.Join(".purr", branchRef)
		hash, err := os.ReadFile(branchPath)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(hash)), nil
	}
	// Case2: Detached HEAD (direct hash)
	return ref, nil
}

// UpdateHEAD updates the current HEAD reference to point to the specified commit hash.
// If HEAD points to a branch (i.e., is a symbolic reference), it updates the branch file
// with the new commit hash. If HEAD is in a detached state, it updates the HEAD file directly.
// Returns an error if reading or writing the reference files fails.
//
// commitHash: The hash of the commit to update HEAD to.
// error: An error if the operation fails, otherwise nil.
func UpdateHEAD(commitHash string) error {
	headPath := filepath.Join(".purr", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return err
	}

	ref := strings.TrimSpace(string(content))

	// Case1: If HEAD points to a branch, update the branch file
	if strings.HasPrefix(ref, "ref:") {
		branchRef := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		branchPath := filepath.Join(".purr", branchRef)
		return os.WriteFile(branchPath, []byte(commitHash+"\n"), 0644)
	}

	// Case2: Detached HEAD - update HEAD directly
	return os.WriteFile(headPath, []byte(commitHash+"\n"), 0644)
}
