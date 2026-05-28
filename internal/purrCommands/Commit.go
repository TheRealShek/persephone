package purrCommands

import (
	"Persephone/internal/ui"
	"Persephone/internal/utils"
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func CommitPurrFiles(path, message, authorName, authorEmail string) error {
	// Get the tree hash (snapshot of current files)
	entries, err := getTreeEntries(path) // Your function to get tree entries
	if err != nil {
		return fmt.Errorf("failed to get tree entries: %w", err)
	}

	// Build tree object for storage
	treeContent, err := utils.BuildTreeObject(entries)
	if err != nil {
		return fmt.Errorf("failed to build tree object: %w", err)
	}

	treeHash, err := utils.ComputeTreeSHA1(entries)
	if err != nil {
		return fmt.Errorf("failed to create tree object: %w", err)
	}

	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(treeContent)
	w.Close()

	err = utils.StoreObject(path, treeHash, compressed.Bytes())
	if err != nil {
		return fmt.Errorf("failed to store tree object: %w", err)
	}

	// Get parent commit hash (empty string if first commit)
	parentHash, err := utils.GetHEADCommit(path)
	if err == nil && parentHash != "" {
		// Get parent commit's tree hash
		parentTreeHash, err := utils.GetCommitTreeHash(path, parentHash)
		if err == nil && parentTreeHash == treeHash {
			return fmt.Errorf("nothing to commit, working tree clean")
		}
	}

	// Create author/committer info
	authorInfo := utils.PurrConfig{
		UserName:  authorName,
		UserEmail: authorEmail,
	}

	// Create the commit object
	commit := &utils.CommitObj{
		TreeHash:   treeHash,
		ParentHash: parentHash,
		Author:     authorInfo,
		Committer:  authorInfo,
		Message:    message,
		Timestamp:  time.Now(),
	}

	// Compute commit hash
	commitHash, err := utils.ComputeCommitSHA1(commit)
	if err != nil {
		return fmt.Errorf("failed to compute commit hash: %w", err)
	}

	// Build and store the commit object
	commitObj, err := utils.BuildCommitObject(commit)
	if err != nil {
		return fmt.Errorf("failed to build commit object: %w", err)
	}

	// Compress with zlib
	compressed.Reset()
	w = zlib.NewWriter(&compressed)
	w.Write(commitObj)
	w.Close()

	// Store the commit object in .purr/objects/{hash[:2]}/{hash[2:]}
	err = utils.StoreObject(path, commitHash, compressed.Bytes())
	if err != nil {
		return fmt.Errorf("failed to store commit object: %w", err)
	}

	// Update HEAD to point to this new commit
	err = utils.UpdateHEAD(path, commitHash)
	if err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}

	fmt.Printf("%s %s\n", ui.Metadata(fmt.Sprintf("[%s]", commitHash[:7])), message)
	return nil
}

// getTreeEntries retrieves the staged file entries from the .purr repository at the given path.
// It performs the following steps:
// 1. Checks if the .purr directory exists at the specified path, returning an error if not.
// 2. Reads the index file from the .purr directory using utils.ReadIndex.
// 3. Returns an error if there are no staged files in the index.
// 4. Converts each index entry to a TreeEntries struct, formatting the SHA-1 and mode appropriately.
// Returns a slice of pointers to TreeEntries and an error if any step fails.

func getTreeEntries(path string) ([]*utils.TreeEntries, error) {
	// Check if .purr directory exists
	purrDir := filepath.Join(path, ".purr")
	if _, err := os.Stat(purrDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a purr repository (or any of the parent directories): .purr")
	}

	// Read the index file
	indexPath := filepath.Join(purrDir, "index")
	index, err := utils.ReadIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	// Check if there are staged files
	if len(index) == 0 {
		return nil, fmt.Errorf("no changes staged for commit")
	}

	// Convert IndexEntry to TreeEntries
	var entries []*utils.TreeEntries
	for _, indexEntry := range index {
		// Convert SHA-1 bytes to hex string
		sha1Hex := fmt.Sprintf("%x", indexEntry.Sha1)

		// Convert mode to Git format string
		mode := getGitMode(indexEntry.Mode)

		// Create tree entry
		entry := &utils.TreeEntries{
			Name:     indexEntry.Path,
			Filename: filepath.Base(indexEntry.Path),
			Sha1Hex:  sha1Hex,
			IsTree:   false,
			Mode:     mode,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// getGitMode returns the Git file mode string ("100755" for executable, "100644" for regular file) based on the file's mode.
func getGitMode(mode uint32) string {
	if mode&0111 != 0 {
		return "100755" // Executable
	}
	return "100644" // Regular file
}
