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

	"Persephone/internal/ui"
)

// BuildTreeObject constructs a Git-compatible tree object from a slice of TreeEntries.
// A Git tree object is a binary snapshot of a directory. Its formatting rules are rigid:
//  - Entries must be sorted lexicographically by name.
//  - In Git sorting conventions, a directory (tree) is compared as if it has a trailing slash "/"
//    (e.g., "src/" sorts after "src.go"), ensuring parent-child folder structures are deterministic.
//  - Format for each entry: "{mode} {name}\x00{20-byte SHA-1 raw bytes}"
//  - The whole content is prepended with the header "tree {size}\x00".
func BuildTreeObject(entries []*TreeEntries) ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries to create tree for")
	}

	// Sort entries according to Git's tree-ordering rules:
	// If it is a directory (sub-tree), we append a virtual "/" for comparison.
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

	var treeContent []byte
	for _, entry := range entries {
		if entry.Mode == "" || entry.Name == "" {
			return nil, fmt.Errorf("invalid entry: mode and name required (got mode='%s', name='%s')", entry.Mode, entry.Name)
		}
		// Validate that the file mode conforms to VCS formats
		if entry.Mode != "100644" && entry.Mode != "100755" && entry.Mode != "040000" {
			return nil, fmt.Errorf("invalid mode for entry %s: %s", entry.Name, entry.Mode)
		}
		
		// Serialize entry header: e.g. "100644 filename.go\x00"
		line := fmt.Sprintf("%s %s\x00", entry.Mode, entry.Name)
		treeContent = append(treeContent, []byte(line)...)
		
		// Decode hex hash into its raw 20-byte SHA-1 binary format
		shaBytes, err := hex.DecodeString(entry.Sha1Hex)
		if err != nil || len(shaBytes) != 20 {
			return nil, fmt.Errorf("invalid SHA-1 for entry %s", entry.Name)
		}
		treeContent = append(treeContent, shaBytes...)
	}

	// Assemble final tree object with metadata header
	header := fmt.Sprintf("tree %d\x00", len(treeContent))
	treeObj := append([]byte(header), treeContent...)

	return treeObj, nil
}

// ComputeCommitSHA1 computes the SHA-1 hash of the serialized commit object.
func ComputeCommitSHA1(commit *CommitObj) (string, error) {
	commitObj, err := BuildCommitObject(commit)
	if err != nil {
		return "", err
	}

	sha := sha1.Sum(commitObj)
	return fmt.Sprintf("%x", sha[:]), nil
}

// BuildCommitObject serializes a CommitObj into the standard Git-compatible commit object layout.
// In Git, a commit object is a plain-text payload containing links to a tree snapshot, parent commit(s),
// author/committer names and timestamps, followed by a double newline and the message:
//
//   commit {size}\x00tree {tree-hash}\n[parent {parent-hash}\n]author {name} <{email}> {unix-time} +0000\n...
//
// This allows tools to parse commits uniformly. Timezones are hardcoded to "+0000" (UTC)
// to guarantee test execution determinism across different developer systems.
func BuildCommitObject(commit *CommitObj) ([]byte, error) {
	if commit.TreeHash == "" {
		return nil, fmt.Errorf("tree hash is required")
	}
	if commit.Message == "" {
		return nil, fmt.Errorf("commit message is required")
	}
	if commit.Author.UserName == "" || commit.Author.UserEmail == "" {
		return nil, fmt.Errorf("author name and email are required")
	}

	var content strings.Builder

	// Write tree pointer
	content.WriteString(fmt.Sprintf("tree %s\n", commit.TreeHash))

	// Link history: Write parent reference (only omitted in the initial root commit)
	if commit.ParentHash != "" {
		content.WriteString(fmt.Sprintf("parent %s\n", commit.ParentHash))
	}

	if commit.Timestamp.IsZero() {
		return nil, fmt.Errorf("commit timestamp is required")
	}
	timestamp := commit.Timestamp.Unix()
	timezone := "+0000" // Standardize on UTC for determinism across environments

	// Write metadata lines
	content.WriteString(fmt.Sprintf("author %s <%s> %d %s\n",
		commit.Author.UserName,
		commit.Author.UserEmail,
		timestamp,
		timezone))

	content.WriteString(fmt.Sprintf("committer %s <%s> %d %s\n",
		commit.Committer.UserName,
		commit.Committer.UserEmail,
		timestamp,
		timezone))

	// Git commits use a single blank line to separate metadata headers from the message body
	content.WriteString("\n")

	content.WriteString(commit.Message)
	if !strings.HasSuffix(commit.Message, "\n") {
		content.WriteString("\n")
	}

	commitContent := []byte(content.String())

	// Build the zlib payload with the binary size header
	header := fmt.Sprintf("commit %d\x00", len(commitContent))
	commitObj := append([]byte(header), commitContent...)

	return commitObj, nil
}

// GetParentCommit resolves the commit hash of the current branch pointed to by HEAD.
// It follows HEAD:
//  1. If HEAD points to a symbolic ref (e.g. "ref: refs/heads/main"), it reads that branch's ref file.
//  2. If the branch file exists, its contents are returned as the parent commit hash.
//  3. If HEAD holds a direct commit SHA-1 (detached state), it returns that SHA-1.
//  4. If no commits exist yet, it returns an empty string, signifying the next commit is the repository root.
func GetParentCommit(repoPath string) (string, error) {
	headPath := filepath.Join(repoPath, ".purr", "HEAD")
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		return "", nil // HEAD doesn't exist yet (uninitialized repo state)
	}

	headStr := strings.TrimSpace(string(headContent))

	// Follow symbolic reference to find branch pointer
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		branchPath := filepath.Join(repoPath, ".purr", refPath)

		parentSHA, err := os.ReadFile(branchPath)
		if err != nil {
			return "", nil // Branch ref is not written yet (first commit on main)
		}

		return strings.TrimSpace(string(parentSHA)), nil
	}

	// Detached HEAD holds a direct SHA-1 hash
	return headStr, nil
}

// UpdateBranchRef updates the active branch's reference file with the newly created commit's SHA-1.
// By writing the 40-character commit hash followed by a newline into `.purr/refs/heads/<branch>`,
// we advance the branch tip to the new commit. Detached HEAD updates are currently not supported.
func UpdateBranchRef(repoPath, commitSHA1 string) error {
	headPath := filepath.Join(repoPath, ".purr", "HEAD")
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		// Bootstrap HEAD if missing
		headContent = []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(headPath, headContent, 0644); err != nil {
			return fmt.Errorf("failed to create HEAD: %w", err)
		}
	}

	headStr := strings.TrimSpace(string(headContent))

	var branchPath string
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		branchPath = filepath.Join(repoPath, ".purr", refPath)
	} else {
		return fmt.Errorf("detached HEAD not supported yet")
	}

	// Ensure the base references directory exists before writing ref update
	dir := filepath.Dir(branchPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create refs directory: %w", err)
	}

	if err := os.WriteFile(branchPath, []byte(commitSHA1+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	return nil
}

// CheckConfigFile validates that author credentials are set in the global config.
// Since commits require author metadata (UserName and UserEmail) to construct valid headers,
// this validation prevents anonymous commits early and guides the user via a friendly hint.
func CheckConfigFile() (string, string, error) {
	config, err := ReadConfig()
	if err != nil {
		return "", "", fmt.Errorf("error reading config: %w", err)
	}

	if config.UserName == "" {
		return "", "", ui.NewHintError(fmt.Errorf("user.name is not set.\nPlease configure it using: purr config user.name \"Your Name\""))
	}

	if config.UserEmail == "" {
		return "", "", ui.NewHintError(fmt.Errorf("user.email is not set.\nPlease configure it using: purr config user.email \"your.email@example.com\""))
	}
	return config.UserName, config.UserEmail, nil
}

// GetCommitTreeHash reads a commit object by its hash and retrieves the root tree SHA-1 it points to.
// This is critical for status checks (detecting if any files are modified compared to the last commit).
// It performs a standard VCS object lookup:
//  1. Reads `.purr/objects/{hash[:2]}/{hash[2:]}`
//  2. Decompresses it via zlib
//  3. Parses the header `commit {size}\x00` and extracts the metadata payload
//  4. Reads the first line which must contain `tree {treeHash}`
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

	// Git binary objects always terminate their type/size header with a null byte
	nullIndex := bytes.IndexByte(content, 0)
	if nullIndex == -1 {
		return "", fmt.Errorf("invalid commit object: no null byte")
	}

	body := content[nullIndex+1:]
	lines := strings.Split(string(body), "\n")

	if len(lines) > 0 && strings.HasPrefix(lines[0], "tree ") {
		treeHash := strings.TrimPrefix(lines[0], "tree ")
		return strings.TrimSpace(treeHash), nil
	}

	return "", fmt.Errorf("invalid commit object: no tree line")
}
