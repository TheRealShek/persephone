package utils

import (
	"Persephone/internal/platform"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ExistsAndIsDirectory checks if the given path exists and represents a directory.
func ExistsAndIsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false, err
	}

	if err != nil {
		return false, fmt.Errorf("stat check failed for %s: %w", path, err)
	}

	return info.IsDir(), nil
}

// WalkAndAddFiles recursively crawls a directory path and executes the handleFile function on each discovered file.
//
// Key VCS Walking Decisions:
//  1. Hidden File Filter: Any file or directory starting with "." is immediately skipped.
//     This is a vital safeguard: it prevents tracking the internal `.purr` repository metadata,
//     developer sandboxes, IDE configuration folders (e.g. `.idea`, `.vscode`), or other Git repositories.
//  2. Directory Skipping: We skip directories after traversing their descendants. Staged records
//     only track actual file contents (blobs); folders are represented implicitly through file paths.
//  3. Fault Tolerance: If `handleFile` fails for a file, the walk logs the error but continues.
//     This ensures that a single locked or unreadable file does not abort a larger concurrent staging operation,
//     improving CLI robustness for bulk file additions.
func WalkAndAddFiles(root string, handleFile func(string) error) error {
	return filepath.WalkDir(root, func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %w", entryPath, err)
		}

		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if err := handleFile(entryPath); err != nil {
			log.Printf("error handling file %s: %v", entryPath, err)
			return nil
		}
		return nil
	})
}

// createdDirs caches fan-out directories that have already been created during this process
// lifetime, eliminating redundant mkdir syscalls (which return EEXIST but still cost a round-trip).
// Each entry stores a *sync.Once to guarantee that concurrent goroutines targeting the same
// 2-char hex prefix block until MkdirAll completes, preventing write-before-mkdir races.
var createdDirs sync.Map

// StoreObject writes compressed object payloads under `.purr/objects/xx/yyyy...`.
// It guarantees that the 2-character hex prefix directory is created before writing the file
// to support object fan-out conventions.
//
// Deduplication: Since object storage is content-addressable, an existing file at the target path
// guarantees byte-identical content. We skip the write entirely to avoid redundant disk I/O.
func StoreObject(rootDir string, hashStr string, data []byte) error {
	objectPath := filepath.Join(rootDir, ".purr", "objects", hashStr[:2], hashStr[2:])
	// Content-addressable dedup: same hash guarantees identical content
	if _, err := os.Stat(objectPath); err == nil {
		return nil
	}

	dir := filepath.Dir(objectPath)
	val, _ := createdDirs.LoadOrStore(dir, &sync.Once{})
	var mkdirErr error
	val.(*sync.Once).Do(func() {
		mkdirErr = os.MkdirAll(dir, 0755)
	})
	if mkdirErr != nil {
		return mkdirErr
	}

	return os.WriteFile(objectPath, data, 0644)
}

// PopulateAllIndexField constructs an IndexEntry from filesystem stat data.
// It leverages platform-specific metadata extractions (to support Unix, Darwin, and Windows inodes/device IDs)
// and maps standard properties into the index entry.
func PopulateAllIndexField(fileInfo os.FileInfo, relPath string) IndexEntry {
	stat := platform.ExtractStat(fileInfo)
	return IndexEntry{
		Ctime: stat.Ctime,
		Mtime: fileInfo.ModTime(),
		Dev:   stat.Dev,
		Ino:   stat.Ino,
		Mode:  uint32(fileInfo.Mode()),
		Uid:   stat.Uid,
		Gid:   stat.Gid,
		Size:  uint32(fileInfo.Size()),
		Stage: 0,
		Path:  relPath,
	}
}

// GetHEADCommit resolves the current HEAD commit hash from `.purr/HEAD`.
// It supports two reference states:
//  1. Symbolic Reference (e.g. `ref: refs/heads/main`): Follows the path to read the active branch's commit hash.
//  2. Detached HEAD State: Reads the raw commit hash directly if HEAD points directly to a commit instead of a branch.
// Returns the resolved 40-character commit hash string.
func GetHEADCommit(rootDir string) (string, error) {
	headPath := filepath.Join(rootDir, ".purr", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	ref := strings.TrimSpace(string(content))

	if strings.HasPrefix(ref, "ref:") {
		branchRef := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		branchPath := filepath.Join(rootDir, ".purr", branchRef)
		hash, err := os.ReadFile(branchPath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		return strings.TrimSpace(string(hash)), nil
	}
	
	return ref, nil
}

// UpdateHEAD advances the HEAD pointer to a new commit hash.
//
// Advancing Invariants:
//  - If HEAD is a symbolic reference (typical branch development), it updates the target branch's reference file,
//    preserving the symbolic link.
//  - If HEAD is in a detached state (direct hash), it overwrites the HEAD file directly.
func UpdateHEAD(rootDir string, commitHash string) error {
	headPath := filepath.Join(rootDir, ".purr", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return err
	}

	ref := strings.TrimSpace(string(content))

	if strings.HasPrefix(ref, "ref:") {
		branchRef := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		branchPath := filepath.Join(rootDir, ".purr", branchRef)
		return os.WriteFile(branchPath, []byte(commitHash+"\n"), 0644)
	}

	return os.WriteFile(headPath, []byte(commitHash+"\n"), 0644)
}

