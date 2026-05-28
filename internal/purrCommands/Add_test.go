package purrCommands_test

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/testutils"
	"Persephone/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

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

	err := purrCommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("AddPurrFiles failed: %v", err)
	}

	entries, err := utils.ReadIndex(filepath.Join(repo, ".purr", "index"))
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

	err := purrCommands.AddPurrFiles(".")
	if err != nil {
		t.Fatalf("Initial AddPurrFiles failed: %v", err)
	}

	indexPath := filepath.Join(repo, ".purr", "index")
	preState, _ := os.ReadFile(indexPath)

	// Inject a broken symlink. os.Stat inside the goroutine will fail.
	badSymlink := filepath.Join(repo, "broken.txt")
	os.Symlink(filepath.Join(repo, "does_not_exist"), badSymlink)

	err = purrCommands.AddPurrFiles(".")
	if err == nil {
		t.Fatalf("Expected AddPurrFiles to fail due to broken symlink, but it succeeded")
	}

	// Verify the error mentions "purr add failed"
	if err != nil && err.Error()[:15] != "purr add failed" {
		t.Errorf("Expected error to start with 'purr add failed', got: %v", err)
	}

	postState, _ := os.ReadFile(indexPath)
	if string(preState) != string(postState) {
		t.Errorf("Index was mutated during a failed add operation")
	}
}
