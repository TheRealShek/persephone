package purrcommands_test

import (
	"persephone/internal/objects"
	"persephone/internal/purrcommands"
	"persephone/internal/refs"
	"persephone/internal/testutils"

	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir changes the working directory to dir and returns a cleanup function
// that restores the original directory. Must be called before AddPurrFiles
// since it relies on os.Getwd().
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

// addAndCommit is a helper that creates test files, stages them, and commits.
// Returns the commit SHA hash read from refs/heads/main.
func addAndCommit(t *testing.T, repo string, files map[string]string, message string) string {
	t.Helper()

	for name, content := range files {
		testutils.WriteTestFile(t, repo, name, content)
	}

	chdir(t, repo)

	if err := purrcommands.AddPurrFiles("."); err != nil {
		t.Fatalf("AddPurrFiles() error = %v", err)
	}

	if err := purrcommands.CommitPurrFiles(repo, message, "Test User", "test@example.com"); err != nil {
		t.Fatalf("CommitPurrFiles() error = %v", err)
	}

	hash, err := refs.GetHEADCommit(repo)
	if err != nil {
		t.Fatalf("GetHEADCommit() error = %v", err)
	}
	return hash
}

func TestCommitPurrFiles_FirstCommit(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	hash := addAndCommit(t, repo, map[string]string{
		"hello.txt": "hello world\n",
	}, "initial commit")

	// HEAD should be a 40-character hex SHA-1
	if len(hash) != 40 {
		t.Errorf("expected 40-char SHA, got %d chars: %q", len(hash), hash)
	}
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("HEAD hash contains invalid hex char %q in %q", string(c), hash)
			break
		}
	}
}

func TestCommitPurrFiles_SecondCommit(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// First commit
	hash1 := addAndCommit(t, repo, map[string]string{
		"file.txt": "version 1\n",
	}, "first commit")

	// Modify the file and commit again
	testutils.WriteTestFile(t, repo, "file.txt", "version 2\n")
	chdir(t, repo)

	if err := purrcommands.AddPurrFiles("."); err != nil {
		t.Fatalf("AddPurrFiles() for second commit error = %v", err)
	}
	if err := purrcommands.CommitPurrFiles(repo, "second commit", "Test User", "test@example.com"); err != nil {
		t.Fatalf("CommitPurrFiles() second commit error = %v", err)
	}

	hash2, err := refs.GetHEADCommit(repo)
	if err != nil {
		t.Fatalf("GetHEADCommit() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("expected HEAD to change after second commit, but it stayed the same")
	}
	if len(hash2) != 40 {
		t.Errorf("expected 40-char SHA for second commit, got %d chars: %q", len(hash2), hash2)
	}
}

func TestCommitPurrFiles_NothingStaged(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// Don't add any files — index is empty
	err := purrcommands.CommitPurrFiles(repo, "empty commit", "Test User", "test@example.com")
	if err == nil {
		t.Fatal("expected error when nothing staged, got nil")
	}
	if !strings.Contains(err.Error(), "nothing to commit") {
		t.Errorf("expected 'nothing to commit' in error, got: %v", err)
	}
}

func TestCommitPurrFiles_CleanTree(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// Create, stage, and commit
	_ = addAndCommit(t, repo, map[string]string{
		"readme.md": "# Hello\n",
	}, "initial commit")

	// Try to commit again without any changes — tree hash should match
	err := purrcommands.CommitPurrFiles(repo, "duplicate commit", "Test User", "test@example.com")
	if err == nil {
		t.Fatal("expected error on clean-tree commit, got nil")
	}
	if !strings.Contains(err.Error(), "nothing to commit") {
		t.Errorf("expected 'nothing to commit' in error, got: %v", err)
	}
}

func TestCommitPurrFiles_CommitObjectStored(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	hash := addAndCommit(t, repo, map[string]string{
		"main.go": "package main\n",
	}, "store commit test")

	// The commit object should exist at .purr/objects/{hash[:2]}/{hash[2:]}
	objPath := filepath.Join(repo, ".purr", "objects", hash[:2], hash[2:])
	info, err := os.Stat(objPath)
	if err != nil {
		t.Fatalf("commit object not found at %s: %v", objPath, err)
	}
	if info.IsDir() {
		t.Errorf("expected commit object to be a file, got directory")
	}
	if info.Size() == 0 {
		t.Errorf("commit object file is empty")
	}
}

func TestCommitPurrFiles_TreeObjectStored(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	hash := addAndCommit(t, repo, map[string]string{
		"data.txt": "some data\n",
	}, "tree object test")

	// Read the commit to extract its tree hash
	treeHash, err := objects.GetCommitTreeHash(repo, hash)
	if err != nil {
		t.Fatalf("GetCommitTreeHash() error = %v", err)
	}

	if len(treeHash) != 40 {
		t.Fatalf("expected 40-char tree hash, got %d chars: %q", len(treeHash), treeHash)
	}

	// The tree object should also be stored in .purr/objects/
	treeObjPath := filepath.Join(repo, ".purr", "objects", treeHash[:2], treeHash[2:])
	info, err := os.Stat(treeObjPath)
	if err != nil {
		t.Fatalf("tree object not found at %s: %v", treeObjPath, err)
	}
	if info.IsDir() {
		t.Errorf("expected tree object to be a file, got directory")
	}
	if info.Size() == 0 {
		t.Errorf("tree object file is empty")
	}
}

func TestCommitPurrFiles_MissingConfig(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "file.txt", "data")

	chdir(t, repo)
	if err := purrcommands.AddPurrFiles("."); err != nil {
		t.Fatalf("AddPurrFiles() error = %v", err)
	}

	// Empty config file to simulate missing user.name
	configPath := filepath.Join(t.TempDir(), ".purrconfig")
	os.WriteFile(configPath, []byte("{}"), 0644)
	t.Setenv("PURR_CONFIG_PATH", configPath)

	// Since we pass author and email as arguments now in CommitPurrFiles signature:
	// wait, CommitPurrFiles signature is CommitPurrFiles(path, message, author, email).
	// If author and email are passed directly, we don't read the config file inside CommitPurrFiles!
	// Let me check how config is handled... Oh, cmd/commit.go reads the config and passes author and email!
	// So CommitPurrFiles doesn't enforce config. I should test that empty string fails.

	err := purrcommands.CommitPurrFiles(repo, "msg", "", "test@example.com")
	if err == nil {
		t.Fatal("expected error with empty author, got nil")
	}

	err = purrcommands.CommitPurrFiles(repo, "msg", "Test User", "")
	if err == nil {
		t.Fatal("expected error with empty email, got nil")
	}
}

func TestCommitPurrFiles_CreatesSubtrees(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// Create nested files
	hash := addAndCommit(t, repo, map[string]string{
		"a/b/c.txt": "nested data\n",
	}, "subtree test")

	// Read root tree hash from commit
	treeHash, err := objects.GetCommitTreeHash(repo, hash)
	if err != nil {
		t.Fatalf("GetCommitTreeHash() error = %v", err)
	}

	// The root tree object should exist and contain an entry for 'a' with mode '040000'
	treeObjPath := filepath.Join(repo, ".purr", "objects", treeHash[:2], treeHash[2:])
	f, err := os.Open(treeObjPath)
	if err != nil {
		t.Fatalf("failed to open root tree object: %v", err)
	}
	defer f.Close()
	r, err := zlib.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create zlib reader: %v", err)
	}
	defer r.Close()
	data, _ := io.ReadAll(r)

	content := string(data)
	if !strings.Contains(content, "040000 a\x00") {
		t.Errorf("expected root tree to contain subtree 'a', got content: %q", content)
	}

	// It should NOT contain the flattened path 'a/b/c.txt'
	if strings.Contains(content, "a/b/c.txt") {
		t.Errorf("expected root tree NOT to contain flattened path, got content: %q", content)
	}
}
