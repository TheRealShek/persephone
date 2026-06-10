package objects

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
	"strconv"
	"strings"
	"time"

	"Persephone/internal/config"
	"Persephone/internal/ui"
)

// BuildTreeObject constructs a Git-compatible tree object from a slice of TreeEntries.
// A Git tree object is a binary snapshot of a directory. Its formatting rules are rigid:
//   - Entries must be sorted lexicographically by name.
//   - In Git sorting conventions, a directory (tree) is compared as if it has a trailing slash "/"
//     (e.g., "src/" sorts after "src.go"), ensuring parent-child folder structures are deterministic.
//   - Format for each entry: "{mode} {name}\x00{20-byte SHA-1 raw bytes}"
//   - The whole content is prepended with the header "tree {size}\x00".
func BuildTreeObject(rootDir string, entries []*TreeEntries) ([]byte, error) {
	if len(entries) == 0 {
		return []byte("tree 0\x00"), nil
	}

	// Group entries by their top-level directory component
	rootFiles := []*TreeEntries{}
	subDirs := make(map[string][]*TreeEntries)

	for _, entry := range entries {
		parts := strings.SplitN(entry.Name, string(filepath.Separator), 2)
		if len(parts) == 1 {
			rootFiles = append(rootFiles, entry)
		} else {
			dirName := parts[0]
			// Update the name to be relative to the subtree
			subEntry := &TreeEntries{
				Name:     parts[1],
				Filename: entry.Filename,
				Sha1Hex:  entry.Sha1Hex,
				IsTree:   entry.IsTree,
				Mode:     entry.Mode,
			}
			subDirs[dirName] = append(subDirs[dirName], subEntry)
		}
	}

	// Recursively build subtrees
	for dirName, subEntries := range subDirs {
		subTreeObj, err := BuildTreeObject(rootDir, subEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to build subtree %s: %w", dirName, err)
		}

		sha := sha1.Sum(subTreeObj)
		subTreeHash := fmt.Sprintf("%x", sha[:])

		var compressed bytes.Buffer
		w := zlib.NewWriter(&compressed)
		if _, err := w.Write(subTreeObj); err != nil {
			return nil, fmt.Errorf("failed to compress subtree %s: %w", dirName, err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("failed to finalize subtree %s: %w", dirName, err)
		}

		if err := StoreObject(rootDir, subTreeHash, compressed.Bytes()); err != nil {
			return nil, fmt.Errorf("failed to store subtree %s: %w", dirName, err)
		}

		// Add subtree reference to the current tree
		rootFiles = append(rootFiles, &TreeEntries{
			Name:     dirName,
			Filename: dirName,
			Sha1Hex:  subTreeHash,
			IsTree:   true,
			Mode:     "040000",
		})
	}

	// Sort entries according to Git's tree-ordering rules:
	// If it is a directory (sub-tree), we append a virtual "/" for comparison.
	sort.Slice(rootFiles, func(i, j int) bool {
		nameI := rootFiles[i].Name
		nameJ := rootFiles[j].Name
		if rootFiles[i].IsTree {
			nameI += "/"
		}
		if rootFiles[j].IsTree {
			nameJ += "/"
		}
		return nameI < nameJ
	})

	var treeContent []byte
	for _, entry := range rootFiles {
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
//	commit {size}\x00tree {tree-hash}\n[parent {parent-hash}\n]author {name} <{email}> {unix-time} +0000\n...
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
	if commit.Committer.UserName == "" || commit.Committer.UserEmail == "" {
		return nil, fmt.Errorf("committer name and email are required")
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
	config, err := config.ReadConfig()
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
	commit, err := ReadCommitObject(rootDir, commitHash)
	if err != nil {
		return "", err
	}

	return commit.TreeHash, nil
}

// ReadCommitObject loads a loose commit object and reconstructs its in-memory representation.
//
// Object Boundary:
// Commits share `.purr/objects` with blobs and trees, so callers must not treat decompressed bytes as
// trusted text. This parser validates the fan-out hash, the `commit <size>\x00` envelope, and the
// metadata required by BuildCommitObject before history traversal follows the parent pointer.
func ReadCommitObject(rootDir string, commitHash string) (*CommitObj, error) {
	if !isSHA1Hex(commitHash) {
		return nil, fmt.Errorf("invalid commit hash %q: expected 40 hexadecimal characters", commitHash)
	}

	objPath := filepath.Join(rootDir, ".purr", "objects", commitHash[:2], commitHash[2:])
	compressed, err := os.ReadFile(objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit object %s: %w", commitHash, err)
	}

	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress commit object %s: %w", commitHash, err)
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed commit object %s: %w", commitHash, err)
	}

	nullIndex := bytes.IndexByte(content, 0)
	if nullIndex == -1 {
		return nil, fmt.Errorf("invalid commit object %s: missing object header terminator", commitHash)
	}

	header := string(content[:nullIndex])
	headerParts := strings.Split(header, " ")
	if len(headerParts) != 2 || headerParts[0] != "commit" {
		return nil, fmt.Errorf("invalid commit object %s: expected commit header", commitHash)
	}

	expectedSize, err := strconv.Atoi(headerParts[1])
	if err != nil || expectedSize < 0 {
		return nil, fmt.Errorf("invalid commit object %s: invalid payload size", commitHash)
	}

	body := content[nullIndex+1:]
	if len(body) != expectedSize {
		return nil, fmt.Errorf("invalid commit object %s: payload size mismatch", commitHash)
	}

	metadata, message, found := strings.Cut(string(body), "\n\n")
	if !found {
		return nil, fmt.Errorf("invalid commit object %s: missing message separator", commitHash)
	}

	commit := &CommitObj{Message: strings.TrimSuffix(message, "\n")}
	var authorTimestamp time.Time

	for _, line := range strings.Split(metadata, "\n") {
		switch {
		case strings.HasPrefix(line, "tree "):
			if commit.TreeHash != "" {
				return nil, fmt.Errorf("invalid commit object %s: duplicate tree header", commitHash)
			}
			commit.TreeHash = strings.TrimPrefix(line, "tree ")
			if !isSHA1Hex(commit.TreeHash) {
				return nil, fmt.Errorf("invalid commit object %s: malformed tree hash", commitHash)
			}
		case strings.HasPrefix(line, "parent "):
			if commit.ParentHash != "" {
				return nil, fmt.Errorf("invalid commit object %s: multiple parents are not supported", commitHash)
			}
			commit.ParentHash = strings.TrimPrefix(line, "parent ")
			if !isSHA1Hex(commit.ParentHash) {
				return nil, fmt.Errorf("invalid commit object %s: malformed parent hash", commitHash)
			}
		case strings.HasPrefix(line, "author "):
			if commit.Author.UserName != "" || commit.Author.UserEmail != "" {
				return nil, fmt.Errorf("invalid commit object %s: duplicate author header", commitHash)
			}
			commit.Author, authorTimestamp, err = parseCommitIdentity(strings.TrimPrefix(line, "author "))
			if err != nil {
				return nil, fmt.Errorf("invalid commit object %s: malformed author: %w", commitHash, err)
			}
		case strings.HasPrefix(line, "committer "):
			if commit.Committer.UserName != "" || commit.Committer.UserEmail != "" {
				return nil, fmt.Errorf("invalid commit object %s: duplicate committer header", commitHash)
			}
			commit.Committer, _, err = parseCommitIdentity(strings.TrimPrefix(line, "committer "))
			if err != nil {
				return nil, fmt.Errorf("invalid commit object %s: malformed committer: %w", commitHash, err)
			}
		default:
			return nil, fmt.Errorf("invalid commit object %s: unsupported metadata header %q", commitHash, line)
		}
	}

	if commit.TreeHash == "" || commit.Author.UserName == "" || commit.Committer.UserName == "" {
		return nil, fmt.Errorf("invalid commit object %s: missing required metadata", commitHash)
	}
	if commit.Message == "" {
		return nil, fmt.Errorf("invalid commit object %s: missing commit message", commitHash)
	}

	commit.Timestamp = authorTimestamp
	return commit, nil
}

// parseCommitIdentity reads the trailing timestamp fields from Git-style identity metadata while
// preserving spaces in developer names. Persephone currently writes UTC offsets and records the
// author timestamp on CommitObj because author and committer timestamps are generated together.
func parseCommitIdentity(value string) (config.PurrConfig, time.Time, error) {
	emailStart := strings.LastIndex(value, " <")
	emailEnd := strings.LastIndex(value, "> ")
	if emailStart <= 0 || emailEnd <= emailStart+2 {
		return config.PurrConfig{}, time.Time{}, fmt.Errorf("expected name <email> timestamp timezone")
	}

	name := value[:emailStart]
	email := value[emailStart+2 : emailEnd]
	timeFields := strings.Fields(value[emailEnd+2:])
	if name == "" || email == "" || len(timeFields) != 2 {
		return config.PurrConfig{}, time.Time{}, fmt.Errorf("expected name <email> timestamp timezone")
	}

	unixTimestamp, err := strconv.ParseInt(timeFields[0], 10, 64)
	if err != nil {
		return config.PurrConfig{}, time.Time{}, fmt.Errorf("invalid unix timestamp")
	}
	if timeFields[1] != "+0000" {
		return config.PurrConfig{}, time.Time{}, fmt.Errorf("unsupported timezone %q", timeFields[1])
	}

	return config.PurrConfig{UserName: name, UserEmail: email}, time.Unix(unixTimestamp, 0).UTC(), nil
}

func isSHA1Hex(hash string) bool {
	if len(hash) != sha1.Size*2 {
		return false
	}

	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}
