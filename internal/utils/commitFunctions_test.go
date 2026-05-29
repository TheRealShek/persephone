package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildTreeObject(t *testing.T) {
	tests := []struct {
		name        string
		entries     []*TreeEntries
		expectError bool
		checkOut    func(t *testing.T, out []byte)
	}{
		{
			name:        "Empty entries",
			entries:     []*TreeEntries{},
			expectError: true,
		},
		{
			name: "Invalid mode",
			entries: []*TreeEntries{
				{Mode: "123456", Name: "file.txt", Sha1Hex: "0123456789abcdef0123456789abcdef01234567"},
			},
			expectError: true,
		},
		{
			name: "Invalid SHA",
			entries: []*TreeEntries{
				{Mode: "100644", Name: "file.txt", Sha1Hex: "invalidhex"},
			},
			expectError: true,
		},
		{
			name: "Valid entries with sorting",
			entries: []*TreeEntries{
				{Mode: "100644", Name: "z-file.txt", Sha1Hex: strings.Repeat("a", 40)},
				{Mode: "040000", Name: "a-dir", IsTree: true, Sha1Hex: strings.Repeat("b", 40)},
			},
			expectError: false,
			checkOut: func(t *testing.T, out []byte) {
				// Header should be tree <size>\0
				if !bytes.HasPrefix(out, []byte("tree ")) {
					t.Errorf("Expected tree header, got: %s", out[:5])
				}
				// Verify sorting (a-dir/ should come before z-file.txt)
				idxA := bytes.Index(out, []byte("040000 a-dir\x00"))
				idxZ := bytes.Index(out, []byte("100644 z-file.txt\x00"))
				if idxA == -1 || idxZ == -1 || idxA > idxZ {
					t.Errorf("Entries not sorted correctly: a-dir idx=%d, z-file idx=%d", idxA, idxZ)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := BuildTreeObject(".", tt.entries)
			if (err != nil) != tt.expectError {
				t.Errorf("BuildTreeObject() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && tt.checkOut != nil {
				tt.checkOut(t, out)
			}
		})
	}
}

func TestBuildCommitObject(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		commit      *CommitObj
		expectError bool
		checkOut    func(t *testing.T, out []byte)
	}{
		{
			name: "Missing Tree Hash",
			commit: &CommitObj{
				Message:   "msg",
				Author:    PurrConfig{UserName: "A", UserEmail: "B"},
				Timestamp: testTime,
			},
			expectError: true,
		},
		{
			name: "Missing Timestamp",
			commit: &CommitObj{
				TreeHash: "treehash",
				Message:  "msg",
				Author:   PurrConfig{UserName: "A", UserEmail: "B"},
			},
			expectError: true,
		},
		{
			name: "Valid commit",
			commit: &CommitObj{
				TreeHash:   "tree123",
				ParentHash: "parent123",
				Author:     PurrConfig{UserName: "Jane Doe", UserEmail: "jane@example.com"},
				Committer:  PurrConfig{UserName: "Jane Doe", UserEmail: "jane@example.com"},
				Message:    "Initial commit",
				Timestamp:  testTime,
			},
			expectError: false,
			checkOut: func(t *testing.T, out []byte) {
				content := string(out)
				if !strings.HasPrefix(content, "commit ") {
					t.Errorf("Missing commit header")
				}
				if !strings.Contains(content, "tree tree123\n") {
					t.Errorf("Missing tree hash line")
				}
				if !strings.Contains(content, "parent parent123\n") {
					t.Errorf("Missing parent hash line")
				}
				timeStr := fmt.Sprintf("%d +0000", testTime.Unix())
				if !strings.Contains(content, timeStr) {
					t.Errorf("Missing correct timestamp")
				}
				if !strings.HasSuffix(content, "\n\nInitial commit\n") {
					t.Errorf("Message formatting incorrect")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := BuildCommitObject(tt.commit)
			if (err != nil) != tt.expectError {
				t.Errorf("BuildCommitObject() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && tt.checkOut != nil {
				tt.checkOut(t, out)
			}
		})
	}
}

// --- New tests below ---

func TestBuildTreeObject_EmptyNameOrMode(t *testing.T) {
	tests := []struct {
		name    string
		entries []*TreeEntries
	}{
		{
			name: "Empty mode",
			entries: []*TreeEntries{
				{Mode: "", Name: "file.txt", Sha1Hex: strings.Repeat("a", 40)},
			},
		},
		{
			name: "Empty name",
			entries: []*TreeEntries{
				{Mode: "100644", Name: "", Sha1Hex: strings.Repeat("a", 40)},
			},
		},
		{
			name: "Both empty",
			entries: []*TreeEntries{
				{Mode: "", Name: "", Sha1Hex: strings.Repeat("a", 40)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildTreeObject(".", tt.entries)
			if err == nil {
				t.Errorf("expected error for entry with empty mode or name, got nil")
			}
		})
	}
}

func TestBuildTreeObject_ExecutableMode(t *testing.T) {
	entries := []*TreeEntries{
		{Mode: "100755", Name: "script.sh", Sha1Hex: strings.Repeat("c", 40)},
	}

	out, err := BuildTreeObject(".", entries)
	if err != nil {
		t.Fatalf("BuildTreeObject() unexpected error for mode 100755: %v", err)
	}

	if !bytes.Contains(out, []byte("100755 script.sh\x00")) {
		t.Errorf("expected executable mode entry in tree object, got: %q", out)
	}
}

func TestBuildTreeObject_DirectoryMode(t *testing.T) {
	entries := []*TreeEntries{
		{Mode: "040000", Name: "subdir", IsTree: true, Sha1Hex: strings.Repeat("d", 40)},
	}

	out, err := BuildTreeObject(".", entries)
	if err != nil {
		t.Fatalf("BuildTreeObject() unexpected error for mode 040000: %v", err)
	}

	if !bytes.Contains(out, []byte("040000 subdir\x00")) {
		t.Errorf("expected directory mode entry in tree object, got: %q", out)
	}

	if !bytes.HasPrefix(out, []byte("tree ")) {
		t.Errorf("expected tree header prefix")
	}
}

func TestComputeCommitSHA1_Deterministic(t *testing.T) {
	testTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	makeCommit := func() *CommitObj {
		return &CommitObj{
			TreeHash:   strings.Repeat("a", 40),
			ParentHash: strings.Repeat("b", 40),
			Author:     PurrConfig{UserName: "Test User", UserEmail: "test@example.com"},
			Committer:  PurrConfig{UserName: "Test User", UserEmail: "test@example.com"},
			Message:    "deterministic test",
			Timestamp:  testTime,
		}
	}

	hash1, err := ComputeCommitSHA1(makeCommit())
	if err != nil {
		t.Fatalf("first ComputeCommitSHA1() error: %v", err)
	}

	hash2, err := ComputeCommitSHA1(makeCommit())
	if err != nil {
		t.Fatalf("second ComputeCommitSHA1() error: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("deterministic check failed: %s != %s", hash1, hash2)
	}

	if len(hash1) != 40 {
		t.Errorf("expected 40-character hex hash, got length %d: %s", len(hash1), hash1)
	}
}

func TestComputeCommitSHA1_DifferentMessages_DifferentHash(t *testing.T) {
	testTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	commitA := &CommitObj{
		TreeHash:  strings.Repeat("a", 40),
		Author:    PurrConfig{UserName: "User", UserEmail: "user@example.com"},
		Committer: PurrConfig{UserName: "User", UserEmail: "user@example.com"},
		Message:   "message alpha",
		Timestamp: testTime,
	}
	commitB := &CommitObj{
		TreeHash:  strings.Repeat("a", 40),
		Author:    PurrConfig{UserName: "User", UserEmail: "user@example.com"},
		Committer: PurrConfig{UserName: "User", UserEmail: "user@example.com"},
		Message:   "message beta",
		Timestamp: testTime,
	}

	hashA, err := ComputeCommitSHA1(commitA)
	if err != nil {
		t.Fatalf("ComputeCommitSHA1(A) error: %v", err)
	}

	hashB, err := ComputeCommitSHA1(commitB)
	if err != nil {
		t.Fatalf("ComputeCommitSHA1(B) error: %v", err)
	}

	if hashA == hashB {
		t.Errorf("expected different hashes for different messages, both got: %s", hashA)
	}
}

// setupPurrDir creates a minimal .purr directory structure for testing.
func setupPurrDir(t *testing.T, root string) {
	t.Helper()
	dirs := []string{
		filepath.Join(root, ".purr"),
		filepath.Join(root, ".purr", "objects"),
		filepath.Join(root, ".purr", "refs", "heads"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", d, err)
		}
	}
}

func TestGetParentCommit_FirstCommit(t *testing.T) {
	root := t.TempDir()
	// No .purr/HEAD at all — should return empty string (first commit)
	parent, err := GetParentCommit(root)
	if err != nil {
		t.Fatalf("GetParentCommit() unexpected error: %v", err)
	}
	if parent != "" {
		t.Errorf("expected empty parent for first commit, got: %q", parent)
	}
}

func TestGetParentCommit_WithExistingCommit(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	commitHash := strings.Repeat("f", 40)

	// Write HEAD pointing to refs/heads/main
	headPath := filepath.Join(root, ".purr", "HEAD")
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("failed to write HEAD: %v", err)
	}

	// Write the branch ref with a commit hash
	refPath := filepath.Join(root, ".purr", "refs", "heads", "main")
	if err := os.WriteFile(refPath, []byte(commitHash+"\n"), 0644); err != nil {
		t.Fatalf("failed to write branch ref: %v", err)
	}

	parent, err := GetParentCommit(root)
	if err != nil {
		t.Fatalf("GetParentCommit() unexpected error: %v", err)
	}
	if parent != commitHash {
		t.Errorf("expected parent %q, got %q", commitHash, parent)
	}
}

func TestGetParentCommit_DetachedHead(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	detachedHash := strings.Repeat("c", 40)

	// Write HEAD as a direct hash (detached HEAD)
	headPath := filepath.Join(root, ".purr", "HEAD")
	if err := os.WriteFile(headPath, []byte(detachedHash+"\n"), 0644); err != nil {
		t.Fatalf("failed to write HEAD: %v", err)
	}

	parent, err := GetParentCommit(root)
	if err != nil {
		t.Fatalf("GetParentCommit() unexpected error: %v", err)
	}
	if parent != detachedHash {
		t.Errorf("expected detached parent %q, got %q", detachedHash, parent)
	}
}

func TestUpdateBranchRef_CreatesRefFile(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	// Write HEAD pointing to refs/heads/main
	headPath := filepath.Join(root, ".purr", "HEAD")
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("failed to write HEAD: %v", err)
	}

	commitHash := strings.Repeat("e", 40)
	if err := UpdateBranchRef(root, commitHash); err != nil {
		t.Fatalf("UpdateBranchRef() unexpected error: %v", err)
	}

	// Verify the ref file was created with the correct content
	refPath := filepath.Join(root, ".purr", "refs", "heads", "main")
	content, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("failed to read branch ref: %v", err)
	}

	expected := commitHash + "\n"
	if string(content) != expected {
		t.Errorf("branch ref content mismatch:\n  got:  %q\n  want: %q", string(content), expected)
	}
}

func TestUpdateBranchRef_NoHEAD(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	// No HEAD file exists — UpdateBranchRef should create HEAD pointing to main
	commitHash := strings.Repeat("d", 40)
	if err := UpdateBranchRef(root, commitHash); err != nil {
		t.Fatalf("UpdateBranchRef() unexpected error: %v", err)
	}

	// Verify HEAD was created pointing to refs/heads/main
	headPath := filepath.Join(root, ".purr", "HEAD")
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("failed to read HEAD after UpdateBranchRef: %v", err)
	}
	if !strings.Contains(string(headContent), "ref: refs/heads/main") {
		t.Errorf("expected HEAD to point to refs/heads/main, got: %q", string(headContent))
	}

	// Verify the ref file was created with the commit hash
	refPath := filepath.Join(root, ".purr", "refs", "heads", "main")
	refContent, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("failed to read branch ref: %v", err)
	}
	expected := commitHash + "\n"
	if string(refContent) != expected {
		t.Errorf("branch ref content mismatch:\n  got:  %q\n  want: %q", string(refContent), expected)
	}
}

func TestGetCommitTreeHash_ValidCommit(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	treeHash := strings.Repeat("a", 40)
	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Build a commit object
	commit := &CommitObj{
		TreeHash:  treeHash,
		Author:    PurrConfig{UserName: "Test", UserEmail: "test@example.com"},
		Committer: PurrConfig{UserName: "Test", UserEmail: "test@example.com"},
		Message:   "test commit",
		Timestamp: testTime,
	}
	commitObj, err := BuildCommitObject(commit)
	if err != nil {
		t.Fatalf("BuildCommitObject() error: %v", err)
	}

	// Compute its hash
	commitSHA := sha1.Sum(commitObj)
	commitHashStr := fmt.Sprintf("%x", commitSHA)

	// Compress and store it in .purr/objects/XX/YYY...
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	if _, err := w.Write(commitObj); err != nil {
		t.Fatalf("zlib write error: %v", err)
	}
	w.Close()

	objDir := filepath.Join(root, ".purr", "objects", commitHashStr[:2])
	if err := os.MkdirAll(objDir, 0755); err != nil {
		t.Fatalf("failed to create object dir: %v", err)
	}
	objPath := filepath.Join(objDir, commitHashStr[2:])
	if err := os.WriteFile(objPath, compressed.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write object file: %v", err)
	}

	// Now test GetCommitTreeHash
	gotTreeHash, err := GetCommitTreeHash(root, commitHashStr)
	if err != nil {
		t.Fatalf("GetCommitTreeHash() unexpected error: %v", err)
	}
	if gotTreeHash != treeHash {
		t.Errorf("tree hash mismatch:\n  got:  %q\n  want: %q", gotTreeHash, treeHash)
	}
}

func TestBuildCommitObject_NoParent(t *testing.T) {
	testTime := time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC)

	commit := &CommitObj{
		TreeHash:  strings.Repeat("a", 40),
		Author:    PurrConfig{UserName: "Author", UserEmail: "author@example.com"},
		Committer: PurrConfig{UserName: "Author", UserEmail: "author@example.com"},
		Message:   "initial commit with no parent",
		Timestamp: testTime,
	}

	out, err := BuildCommitObject(commit)
	if err != nil {
		t.Fatalf("BuildCommitObject() unexpected error: %v", err)
	}

	content := string(out)

	// Should NOT contain a "parent" line
	if strings.Contains(content, "parent ") {
		t.Errorf("expected no parent line in commit object, but found one:\n%s", content)
	}

	// Should still contain tree line
	if !strings.Contains(content, "tree "+strings.Repeat("a", 40)+"\n") {
		t.Errorf("missing tree line in commit object")
	}

	// Should contain message
	if !strings.HasSuffix(content, "\n\ninitial commit with no parent\n") {
		t.Errorf("message formatting incorrect:\n%s", content)
	}
}

func TestBuildCommitObject_MissingMessage(t *testing.T) {
	testTime := time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC)

	commit := &CommitObj{
		TreeHash:  strings.Repeat("a", 40),
		Message:   "", // empty message
		Author:    PurrConfig{UserName: "User", UserEmail: "user@example.com"},
		Committer: PurrConfig{UserName: "User", UserEmail: "user@example.com"},
		Timestamp: testTime,
	}

	_, err := BuildCommitObject(commit)
	if err == nil {
		t.Errorf("expected error for empty message, got nil")
	}
}

func TestBuildCommitObject_MissingAuthor(t *testing.T) {
	testTime := time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		commit *CommitObj
	}{
		{
			name: "Empty author name",
			commit: &CommitObj{
				TreeHash:  strings.Repeat("a", 40),
				Message:   "test",
				Author:    PurrConfig{UserName: "", UserEmail: "user@example.com"},
				Committer: PurrConfig{UserName: "User", UserEmail: "user@example.com"},
				Timestamp: testTime,
			},
		},
		{
			name: "Empty author email",
			commit: &CommitObj{
				TreeHash:  strings.Repeat("a", 40),
				Message:   "test",
				Author:    PurrConfig{UserName: "User", UserEmail: ""},
				Committer: PurrConfig{UserName: "User", UserEmail: "user@example.com"},
				Timestamp: testTime,
			},
		},
		{
			name: "Both author fields empty",
			commit: &CommitObj{
				TreeHash:  strings.Repeat("a", 40),
				Message:   "test",
				Author:    PurrConfig{},
				Committer: PurrConfig{UserName: "User", UserEmail: "user@example.com"},
				Timestamp: testTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildCommitObject(tt.commit)
			if err == nil {
				t.Errorf("expected error for missing author info, got nil")
			}
		})
	}
}
