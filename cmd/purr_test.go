package cmd_test

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/testutils"
	"Persephone/internal/utils"
	"os"
	"path/filepath"
	"testing"
)

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
