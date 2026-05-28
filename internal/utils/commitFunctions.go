package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

/*
BuildTreeObject constructs a Git tree object from a slice of TreeEntries.
It sorts the entries (directories before files, lexicographically), validates each entry,
and encodes them in the Git tree object format:
  - Each entry is "{mode} {name}\0{20-byte SHA-1}"

Returns the raw bytes of the tree object, or an error if validation fails.
*/
func BuildTreeObject(entries []*TreeEntries) ([]byte, error) {
	// Check for empty entries list
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries to create tree for")
	}

	// Sort entries (directories as "name/", files as "name")
	sort.Slice(entries, func(i, j int) bool {
		nameI := entries[i].Name
		nameJ := entries[j].Name
		if entries[i].IsTree {
			nameI += "/"
		}
		if entries[j].IsTree {
			nameJ += "/"
		}
		return nameI < nameJ
	})

	// Build tree content
	var treeContent []byte
	for _, entry := range entries {
		// Validation: mode and name must not be empty
		if entry.Mode == "" || entry.Name == "" {
			return nil, fmt.Errorf("invalid entry: mode and name required (got mode='%s', name='%s')", entry.Mode, entry.Name)
		}
		// Validation: mode must be a valid Git mode
		if entry.Mode != "100644" && entry.Mode != "100755" && entry.Mode != "040000" {
			return nil, fmt.Errorf("invalid mode for entry %s: %s", entry.Name, entry.Mode)
		}
		// {mode} {name}\0
		line := fmt.Sprintf("%s %s\x00", entry.Mode, entry.Name)
		treeContent = append(treeContent, []byte(line)...)
		// 20 raw bytes of SHA-1 (decode hex)
		shaBytes, err := hex.DecodeString(entry.Sha1Hex)
		if err != nil || len(shaBytes) != 20 {
			return nil, fmt.Errorf("invalid SHA-1 for entry %s", entry.Name)
		}
		treeContent = append(treeContent, shaBytes...)
	}

	// Create tree object
	header := fmt.Sprintf("tree %d\x00", len(treeContent))
	treeObj := append([]byte(header), treeContent...)

	return treeObj, nil
}

// ComputeCommitSHA1 computes the SHA-1 of a Git-compatible commit object.
// Similar to ComputeTreeSHA1, but for commit objects.
func ComputeCommitSHA1(commit *CommitObj) (string, error) {
	commitObj, err := BuildCommitObject(commit)
	if err != nil {
		return "", err
	}

	sha := sha1.Sum(commitObj)
	return fmt.Sprintf("%x", sha[:]), nil
}

// BuildCommitObject creates a Git-compatible commit object from a CommitObj struct.
// Format:
// commit {size}\0tree {tree-hash}\nparent {parent-hash}\nauthor {name} <{email}> {timestamp} {timezone}\ncommitter {name} <{email}> {timestamp} {timezone}\n\n{message}\n
func BuildCommitObject(commit *CommitObj) ([]byte, error) {
	// Validation
	if commit.TreeHash == "" {
		return nil, fmt.Errorf("tree hash is required")
	}
	if commit.Message == "" {
		return nil, fmt.Errorf("commit message is required")
	}
	if commit.Author.UserName == "" || commit.Author.UserEmail == "" {
		return nil, fmt.Errorf("author name and email are required")
	}

	// Build commit content
	var content strings.Builder

	// Tree line
	content.WriteString(fmt.Sprintf("tree %s\n", commit.TreeHash))

	// Parent line (if exists)
	if commit.ParentHash != "" {
		content.WriteString(fmt.Sprintf("parent %s\n", commit.ParentHash))
	}

	// Timestamp
	if commit.Timestamp.IsZero() {
		return nil, fmt.Errorf("commit timestamp is required")
	}
	timestamp := commit.Timestamp.Unix()
	timezone := "+0000" // UTC

	// Author line
	content.WriteString(fmt.Sprintf("author %s <%s> %d %s\n",
		commit.Author.UserName,
		commit.Author.UserEmail,
		timestamp,
		timezone))

	// Committer line
	content.WriteString(fmt.Sprintf("committer %s <%s> %d %s\n",
		commit.Committer.UserName,
		commit.Committer.UserEmail,
		timestamp,
		timezone))

	// Empty line before message
	content.WriteString("\n")

	// Commit message
	content.WriteString(commit.Message)
	if !strings.HasSuffix(commit.Message, "\n") {
		content.WriteString("\n")
	}

	commitContent := []byte(content.String())

	// Create commit object with header
	header := fmt.Sprintf("commit %d\x00", len(commitContent))
	commitObj := append([]byte(header), commitContent...)

	return commitObj, nil
}

// Helper: Get parent commit SHA-1 from current branch
func GetParentCommit(repoPath string) (string, error) {
	// Read HEAD to find current branch
	headPath := filepath.Join(repoPath, ".purr", "HEAD")
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		return "", nil // No HEAD yet (first commit)
	}

	headStr := strings.TrimSpace(string(headContent))

	// Parse "ref: refs/heads/main"
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		branchPath := filepath.Join(repoPath, ".purr", refPath)

		parentSHA, err := os.ReadFile(branchPath)
		if err != nil {
			return "", nil // Branch exists but no commits yet
		}

		return strings.TrimSpace(string(parentSHA)), nil
	}

	// Detached HEAD case
	return headStr, nil
}

// Helper: Update branch reference with new commit
func UpdateBranchRef(repoPath, commitSHA1 string) error {
	// Read HEAD to find current branch
	headPath := filepath.Join(repoPath, ".purr", "HEAD")
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		// No HEAD, create it pointing to main
		headContent = []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(headPath, headContent, 0644); err != nil {
			return fmt.Errorf("failed to create HEAD: %w", err)
		}
	}

	headStr := strings.TrimSpace(string(headContent))

	// Parse "ref: refs/heads/main"
	var branchPath string
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		branchPath = filepath.Join(repoPath, ".purr", refPath)
	} else {
		return fmt.Errorf("detached HEAD not supported yet")
	}

	// Create refs/heads directory if needed
	dir := filepath.Dir(branchPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create refs directory: %w", err)
	}

	// Write commit SHA-1 to branch file
	if err := os.WriteFile(branchPath, []byte(commitSHA1+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	return nil
}

// CheckConfigFile verifies that the user's name and email are set in the configuration file.
// It returns an error if either user.name or user.email is missing.
func CheckConfigFile() (string, string, error) {
	// Read config to get user.name and user.email
	config, err := ReadConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return "", "", err
	}

	// Check if user.name is set
	if config.UserName == "" {
		fmt.Println("Error: user.name is not set.")
		fmt.Println("Please configure it using: purr config user.name \"Your Name\"")
		return "", "", err
	}

	// Check if user.email is set
	if config.UserEmail == "" {
		fmt.Println("Error: user.email is not set.")
		fmt.Println("Please configure it using: purr config user.email \"your.email@example.com\"")
		return "", "", err
	}
	return config.UserName, config.UserEmail, nil
}

// GetCommitTreeHash reads a commit object by its hash and extracts the tree hash it references.
// The commit object is expected to be stored in .purr/objects/{first2}/{rest} (zlib-compressed).
// It decompresses the object, skips the header ("commit <size>\0"), and parses the first line
// to find the "tree <hash>" entry. Returns the tree hash string, or an error if not found or invalid.
func GetCommitTreeHash(rootDir string, commitHash string) (string, error) {
	objPath := filepath.Join(rootDir, ".purr", "objects", commitHash[:2], commitHash[2:])
	compressed, err := os.ReadFile(objPath)
	if err != nil {
		return "", err
	}

	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", err
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	// Skip header: "commit <size>\0"
	nullIndex := bytes.IndexByte(content, 0)
	if nullIndex == -1 {
		return "", fmt.Errorf("invalid commit object: no null byte")
	}

	// Parse content after null byte
	body := content[nullIndex+1:]
	lines := strings.Split(string(body), "\n")

	if len(lines) > 0 && strings.HasPrefix(lines[0], "tree ") {
		treeHash := strings.TrimPrefix(lines[0], "tree ")
		return strings.TrimSpace(treeHash), nil
	}

	return "", fmt.Errorf("invalid commit object: no tree line")
}
