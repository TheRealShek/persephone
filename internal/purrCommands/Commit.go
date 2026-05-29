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

// CommitPurrFiles constructs a new commit snapshot from the currently staged files in the index.
//
// Staging Snapshot Lifecycle:
//  1. Snapshot: We load the flat entries from the index and construct a Git-compatible tree object.
//  2. Object Storage: The tree object is zlib-compressed and stored under `.purr/objects` using its SHA-1 hash.
//  3. Change Detection: To prevent empty commits, we look up the parent commit from HEAD. We extract the parent's
//     root tree hash and compare it with the current tree hash. If they are identical, no files have changed,
//     and we abort with "nothing to commit, working tree clean".
//  4. Link History: A commit object is built containing tree pointer, parent pointer, metadata (UserName, UserEmail,
//     and the timestamp), and commit message. This object is zlib-compressed and stored.
//  5. Move HEAD: We advance the branch ref symbolically linked by HEAD to point to the new commit hash, advancing the branch pointer.
func CommitPurrFiles(path, message, authorName, authorEmail string) error {
	entries, err := getTreeEntries(path)
	if err != nil {
		return fmt.Errorf("failed to get tree entries: %w", err)
	}

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

	// Prevent empty commits: check if tree hash matches parent tree hash
	parentHash, err := utils.GetHEADCommit(path)
	if err == nil && parentHash != "" {
		parentTreeHash, err := utils.GetCommitTreeHash(path, parentHash)
		if err == nil && parentTreeHash == treeHash {
			return fmt.Errorf("nothing to commit, working tree clean")
		}
	}

	authorInfo := utils.PurrConfig{
		UserName:  authorName,
		UserEmail: authorEmail,
	}

	commit := &utils.CommitObj{
		TreeHash:   treeHash,
		ParentHash: parentHash,
		Author:     authorInfo,
		Committer:  authorInfo,
		Message:    message,
		Timestamp:  time.Now(),
	}

	commitHash, err := utils.ComputeCommitSHA1(commit)
	if err != nil {
		return fmt.Errorf("failed to compute commit hash: %w", err)
	}

	commitObj, err := utils.BuildCommitObject(commit)
	if err != nil {
		return fmt.Errorf("failed to build commit object: %w", err)
	}

	compressed.Reset()
	w = zlib.NewWriter(&compressed)
	w.Write(commitObj)
	w.Close()

	err = utils.StoreObject(path, commitHash, compressed.Bytes())
	if err != nil {
		return fmt.Errorf("failed to store commit object: %w", err)
	}

	err = utils.UpdateHEAD(path, commitHash)
	if err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}

	// Display short 7-character commit hash prefix, matching standard VCS developer layouts
	fmt.Printf("%s %s\n", ui.Metadata(fmt.Sprintf("[%s]", commitHash[:7])), message)
	return nil
}

// getTreeEntries resolves and parses index staged records for the commit builder.
// It maps the flat list of files staged in `.purr/index` into a slice of `TreeEntries`.
func getTreeEntries(path string) ([]*utils.TreeEntries, error) {
	purrDir := filepath.Join(path, ".purr")
	if _, err := os.Stat(purrDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a purr repository (or any of the parent directories): .purr")
	}

	indexPath := filepath.Join(purrDir, "index")
	index, err := utils.ReadIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	if len(index) == 0 {
		return nil, fmt.Errorf("no changes staged for commit")
	}

	var entries []*utils.TreeEntries
	for _, indexEntry := range index {
		sha1Hex := fmt.Sprintf("%x", indexEntry.Sha1)
		mode := getGitMode(indexEntry.Mode)

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

// getGitMode maps standard filesystem permission bits to standard Git string representations.
// "100755" is used for executable files, and "100644" for regular files.
func getGitMode(mode uint32) string {
	if mode&0111 != 0 {
		return "100755"
	}
	return "100644"
}
