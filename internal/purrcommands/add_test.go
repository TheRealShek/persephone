package purrcommands_test

import (
	"persephone/internal/index"
	"persephone/internal/purrcommands"
	"persephone/internal/testutils"

	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// ---------- Original tests (preserved) ----------

func TestAddAllPurrFiles_ConcurrencyStress(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// Create 100 files to stress the worker pool (capped at 5 * CPU)
	for i := 0; i < 100; i++ {
		testutils.WriteTestFile(t, repo, fmt.Sprintf("file_%d.txt", i), fmt.Sprintf("data %d", i))
	}

	// Change dir to repo (since AddPurrFiles uses os.Getwd())
	// For testing the internal func addAllPurrFiles directly, we just pass the path.
	// We have to export addAllPurrFiles, or we can test AddPurrFiles by setting wd.
	// Since AddPurrFiles uses Getwd(), we will change working directory.
	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("AddPurrFiles failed: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	if len(entries) != 100 {
		t.Errorf("Expected 100 files in index, got %d", len(entries))
	}
}

func TestAddAllPurrFiles_AbortsOnFailure_PreservesIndex(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// Pre-existing valid file
	testutils.WriteTestFile(t, repo, "valid.txt", "valid")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("Initial AddPurrFiles failed: %v", err)
	}

	indexPath := filepath.Join(repo, ".purr", "index")
	preState, _ := os.ReadFile(indexPath)

	// Inject a broken symlink. os.Stat inside the goroutine will fail.
	badSymlink := filepath.Join(repo, "broken.txt")
	os.Symlink(filepath.Join(repo, "does_not_exist"), badSymlink)

	err = purrcommands.AddPurrFiles(".")
	if err == nil {
		t.Fatalf("Expected AddPurrFiles to fail due to broken symlink, but it succeeded")
	}

	// Verify the error mentions "purr add completed"
	if len(err.Error()) < 18 || err.Error()[:18] != "purr add completed" {
		t.Errorf("Expected error to start with 'purr add completed', got: %v", err)
	}

	postState, _ := os.ReadFile(indexPath)
	if string(preState) != string(postState) {
		t.Errorf("Index was mutated during a failed add operation")
	}
}

// ---------- New tests ----------

func TestAddSpecificFiles_SingleFile(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "hello.txt", "hello world")
	testutils.WriteTestFile(t, repo, "other.txt", "should not be staged")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles("hello.txt")
	if err != nil {
		t.Fatalf("AddPurrFiles(\"hello.txt\") failed: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 file in index, got %d", len(entries))
	}

	if entries[0].Path != "hello.txt" {
		t.Errorf("Expected path 'hello.txt', got %q", entries[0].Path)
	}
}

func TestAddSpecificFiles_HiddenFileSkipped(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, ".hidden", "secret")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	// Adding a hidden file by name should skip it
	err := purrcommands.AddPurrFiles(".hidden")
	if err != nil {
		t.Fatalf("AddPurrFiles should not fail for hidden files, got: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Hidden files should not be added to index, but got %d entries", len(entries))
	}
}

func TestAddSpecificFiles_DirectoryRejected(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	dirPath := filepath.Join(repo, "subdir")
	os.MkdirAll(dirPath, 0755)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles("subdir")
	if err == nil {
		t.Fatal("Expected error when adding a directory by name, but got nil")
	}
}

func TestAddPurrFiles_NoPurrDirectory(t *testing.T) {
	// Create a directory without .purr initialization
	tmpDir := t.TempDir()

	originalWD, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles(".")
	if err == nil {
		t.Fatal("Expected error when .purr doesn't exist, but got nil")
	}
}

func TestAddPurrFiles_NoArgs(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := purrcommands.AddPurrFiles()

	w.Close()
	os.Stdout = stdout
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("AddPurrFiles() with no args should not error, got: %v", err)
	}

	if !bytes.Contains(output, []byte("[WARNING]")) || !bytes.Contains(output, []byte("No files selected to add")) {
		t.Fatalf("expected warning-style empty add message, got %q", string(output))
	}
}

func TestAddAllFiles_SkipsHiddenDirectories(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "visible.txt", "I'm visible")
	testutils.WriteTestFile(t, repo, ".hidden_dir/secret.txt", "I'm hidden")
	testutils.WriteTestFile(t, repo, ".dotfile", "dot")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("AddPurrFiles(\".\") failed: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	// Only visible.txt should be indexed. Hidden dir and dotfile should be skipped.
	if len(entries) != 1 {
		t.Errorf("Expected 1 file in index (visible.txt only), got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Path != "visible.txt" {
		t.Errorf("Expected 'visible.txt', got %q", entries[0].Path)
	}
}

func TestAddAllFiles_IdempotentReAdd(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "file.txt", "data")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	// Add once
	err := purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("First AddPurrFiles failed: %v", err)
	}

	indexPath := filepath.Join(repo, ".purr", "index")
	entries1, _ := index.ReadIndex(indexPath)

	// Add again without changes
	err = purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("Second AddPurrFiles failed: %v", err)
	}

	entries2, _ := index.ReadIndex(indexPath)

	if len(entries1) != len(entries2) {
		t.Errorf("Re-adding unchanged files changed entry count: %d -> %d", len(entries1), len(entries2))
	}

	// SHA should remain the same
	if len(entries1) > 0 && len(entries2) > 0 {
		if entries1[0].Sha1 != entries2[0].Sha1 {
			t.Errorf("SHA1 changed on re-add without file modification")
		}
	}
}

func TestAddSpecificFiles_MultipleFiles(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "a.txt", "alpha")
	testutils.WriteTestFile(t, repo, "b.txt", "beta")
	testutils.WriteTestFile(t, repo, "c.txt", "gamma")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles("a.txt", "c.txt")
	if err != nil {
		t.Fatalf("AddPurrFiles with multiple files failed: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 files in index, got %d", len(entries))
	}

	// Entries should be sorted
	if entries[0].Path != "a.txt" {
		t.Errorf("Expected first entry 'a.txt', got %q", entries[0].Path)
	}
	if entries[1].Path != "c.txt" {
		t.Errorf("Expected second entry 'c.txt', got %q", entries[1].Path)
	}
}

func TestAddSpecificFiles_NonExistentFile(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles("does_not_exist.txt")
	if err != nil {
		t.Fatalf("Expected nil when adding non-existent file (should unstage), got error: %v", err)
	}

	// Verify index is empty
	entries, _ := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after unstaging non-existent file, got %d", len(entries))
	}
}

func TestAddAllFiles_NestedDirectories(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "root.txt", "root")
	testutils.WriteTestFile(t, repo, "sub/nested.txt", "nested")
	testutils.WriteTestFile(t, repo, "sub/deep/deep.txt", "deep")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("AddPurrFiles(\".\") failed: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 files in index, got %d", len(entries))
	}

	// Verify paths are relative and sorted
	expectedPaths := []string{"root.txt", "sub/deep/deep.txt", "sub/nested.txt"}
	for i, expected := range expectedPaths {
		if i < len(entries) && entries[i].Path != expected {
			t.Errorf("Entry %d: expected path %q, got %q", i, expected, entries[i].Path)
		}
	}
}

func TestAddAllFiles_BlobObjectCreated(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "file.txt", "content")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	err := purrcommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("AddPurrFiles failed: %v", err)
	}

	entries, _ := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) == 0 {
		t.Fatal("No entries in index")
	}

	// Verify the blob object file exists
	sha1Hex := fmt.Sprintf("%x", entries[0].Sha1)
	objPath := filepath.Join(repo, ".purr", "objects", sha1Hex[:2], sha1Hex[2:])
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		t.Errorf("Blob object was not created at %s", objPath)
	}
}

func TestAddAllFiles_DetectsDeletions(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	filePath := testutils.WriteTestFile(t, repo, "file.txt", "data")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	if err := purrcommands.AddPurrFiles("."); err != nil {
		t.Fatalf("First add failed: %v", err)
	}

	// Verify it's in the index
	entries, _ := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	// Delete from disk
	os.Remove(filePath)

	// Add again
	if err := purrcommands.AddPurrFiles("."); err != nil {
		t.Fatalf("Second add failed: %v", err)
	}

	// Verify index is now empty
	entries, _ = index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after deletion, got %d", len(entries))
	}
}

func TestAddSpecificFiles_UnstageDeletedFile(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	filePath := testutils.WriteTestFile(t, repo, "file.txt", "data")

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	if err := purrcommands.AddPurrFiles("file.txt"); err != nil {
		t.Fatalf("First add failed: %v", err)
	}

	// Verify it's in the index
	entries, _ := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	// Delete from disk
	os.Remove(filePath)

	// Add the specific file again
	if err := purrcommands.AddPurrFiles("file.txt"); err != nil {
		t.Fatalf("Second add failed: %v", err)
	}

	// Verify index is now empty
	entries, _ = index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after specific deletion, got %d", len(entries))
	}
}
