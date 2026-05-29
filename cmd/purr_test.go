package cmd_test

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/testutils"
	"Persephone/internal/utils"
	"os"
	"path/filepath"
	"testing"
)

// ---------- Original test (preserved) ----------

func TestFullWorkflow_InitAddCommit(t *testing.T) {
	// We want an isolated directory, NOT a pre-initialized one,
	// so we create it manually instead of using SetupTestRepo (which does Init).
	repo := t.TempDir()

	// 1. Init
	err := purrCommands.InitPurrDirectories(repo)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify .purr exists
	purrDir := filepath.Join(repo, ".purr")
	if _, err := os.Stat(purrDir); os.IsNotExist(err) {
		t.Fatalf(".purr directory was not created")
	}

	// Setup config
	configPath := filepath.Join(repo, ".purrconfig")
	configContent := `{"user_name":"Test E2E","user_email":"e2e@example.com"}`
	os.WriteFile(configPath, []byte(configContent), 0644)
	t.Setenv("PURR_CONFIG_PATH", configPath)

	// Change working directory for Add (since it uses Getwd)
	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	// 2. Create files
	testutils.WriteTestFile(t, repo, "file1.txt", "hello world")
	testutils.WriteTestFile(t, repo, "dir/file2.txt", "nested")

	// 3. Add files
	err = purrCommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify Index
	entries, _ := utils.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 2 {
		t.Fatalf("Expected 2 files in index, got %d", len(entries))
	}

	// 4. Commit
	err = purrCommands.CommitPurrFiles(repo, "First commit", "Test E2E", "e2e@example.com")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify HEAD is updated
	headCommit, err := utils.GetHEADCommit(repo)
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}
	if len(headCommit) != 40 {
		t.Errorf("Expected 40-character SHA-1 in HEAD, got %q", headCommit)
	}

	// Verify Commit object exists in objects directory
	commitObjPath := filepath.Join(repo, ".purr", "objects", headCommit[:2], headCommit[2:])
	if _, err := os.Stat(commitObjPath); os.IsNotExist(err) {
		t.Errorf("Commit object file was not written to %s", commitObjPath)
	}
}

// ---------- New E2E tests ----------

func TestFullWorkflow_DoubleCommit(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	// Create and add initial file
	testutils.WriteTestFile(t, repo, "file.txt", "version 1")
	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("First add failed: %v", err)
	}

	// First commit
	err := purrCommands.CommitPurrFiles(repo, "First commit", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	firstHead, err := utils.GetHEADCommit(repo)
	if err != nil {
		t.Fatalf("Failed to read HEAD after first commit: %v", err)
	}

	// Modify file, re-add, and commit again
	testutils.WriteTestFile(t, repo, "file.txt", "version 2")
	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("Second add failed: %v", err)
	}

	err = purrCommands.CommitPurrFiles(repo, "Second commit", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("Second commit failed: %v", err)
	}

	secondHead, err := utils.GetHEADCommit(repo)
	if err != nil {
		t.Fatalf("Failed to read HEAD after second commit: %v", err)
	}

	// HEAD should have changed
	if firstHead == secondHead {
		t.Error("HEAD did not change between commits")
	}

	// Both commit objects should exist
	for _, hash := range []string{firstHead, secondHead} {
		objPath := filepath.Join(repo, ".purr", "objects", hash[:2], hash[2:])
		if _, err := os.Stat(objPath); os.IsNotExist(err) {
			t.Errorf("Commit object %s does not exist at %s", hash[:7], objPath)
		}
	}
}

func TestFullWorkflow_CleanTreeRejectsCommit(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	testutils.WriteTestFile(t, repo, "file.txt", "data")
	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// First commit should succeed
	err := purrCommands.CommitPurrFiles(repo, "Initial", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	// Second commit without changes should fail with "nothing to commit"
	err = purrCommands.CommitPurrFiles(repo, "Duplicate", "Test User", "test@example.com")
	if err == nil {
		t.Fatal("Expected error for clean tree commit, but got nil")
	}
}

func TestFullWorkflow_AddThenListFiles(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	testutils.WriteTestFile(t, repo, "alpha.txt", "a")
	testutils.WriteTestFile(t, repo, "beta.txt", "b")

	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// ListFiles should not error
	err := purrCommands.ListFiles(repo, false)
	if err != nil {
		t.Fatalf("ListFiles (normal mode) failed: %v", err)
	}

	err = purrCommands.ListFiles(repo, true)
	if err != nil {
		t.Fatalf("ListFiles (debug mode) failed: %v", err)
	}
}

func TestFullWorkflow_AddSpecificThenAll(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	testutils.WriteTestFile(t, repo, "first.txt", "1")
	testutils.WriteTestFile(t, repo, "second.txt", "2")
	testutils.WriteTestFile(t, repo, "third.txt", "3")

	// Add only one file first
	if err := purrCommands.AddPurrFiles("first.txt"); err != nil {
		t.Fatalf("Specific add failed: %v", err)
	}

	entries, _ := utils.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 1 {
		t.Fatalf("Expected 1 file after specific add, got %d", len(entries))
	}

	// Now add all
	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("Add all failed: %v", err)
	}

	entries, _ = utils.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 3 {
		t.Fatalf("Expected 3 files after add all, got %d", len(entries))
	}
}

func TestFullWorkflow_InitIdempotent(t *testing.T) {
	repo := t.TempDir()

	// Init twice should not fail
	err := purrCommands.InitPurrDirectories(repo)
	if err != nil {
		t.Fatalf("First init failed: %v", err)
	}

	err = purrCommands.InitPurrDirectories(repo)
	if err != nil {
		t.Fatalf("Second init failed: %v", err)
	}

	// Verify structure is intact
	purrDir := filepath.Join(repo, ".purr")
	requiredPaths := []string{
		filepath.Join(purrDir, "objects"),
		filepath.Join(purrDir, "refs", "heads"),
		filepath.Join(purrDir, "logs"),
		filepath.Join(purrDir, "index"),
		filepath.Join(purrDir, "HEAD"),
	}

	for _, p := range requiredPaths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("Required path missing after double init: %s", p)
		}
	}
}

func TestFullWorkflow_CommitWithNewFile(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	originalWD, _ := os.Getwd()
	os.Chdir(repo)
	defer os.Chdir(originalWD)

	// First commit with one file
	testutils.WriteTestFile(t, repo, "original.txt", "original")
	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("First add failed: %v", err)
	}
	if err := purrCommands.CommitPurrFiles(repo, "First", "Test User", "test@example.com"); err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	firstHead, _ := utils.GetHEADCommit(repo)

	// Second commit adding a new file (original unchanged)
	testutils.WriteTestFile(t, repo, "new_file.txt", "new content")
	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("Second add failed: %v", err)
	}
	if err := purrCommands.CommitPurrFiles(repo, "Second", "Test User", "test@example.com"); err != nil {
		t.Fatalf("Second commit failed: %v", err)
	}

	secondHead, _ := utils.GetHEADCommit(repo)

	if firstHead == secondHead {
		t.Error("Adding a new file should produce a different commit hash")
	}

	// Verify index has 2 entries
	entries, _ := utils.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries in index, got %d", len(entries))
	}
}
