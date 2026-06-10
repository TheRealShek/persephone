package purrcommands_test

import (
	"os"
	"path/filepath"
	"testing"

	"persephone/internal/index"
	"persephone/internal/purrcommands"
	"persephone/internal/testutils"
)

func TestRemovePurrFiles_RemovesTrackedFile(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	filePath := testutils.WriteTestFile(t, repo, "keep.txt", "content")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	if err := purrcommands.AddPurrFiles("keep.txt"); err != nil {
		t.Fatalf("AddPurrFiles failed: %v", err)
	}

	if err := purrcommands.RemovePurrFiles("keep.txt"); err != nil {
		t.Fatalf("RemovePurrFiles failed: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed from disk, got err=%v", filePath, err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty index after removal, got %d entries", len(entries))
	}
}

func TestRemovePurrFiles_RejectsUntrackedFile(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "ghost.txt", "content")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	if err := purrcommands.RemovePurrFiles("ghost.txt"); err == nil {
		t.Fatal("expected error removing untracked file, got nil")
	}

	if _, err := os.Stat(filepath.Join(repo, "ghost.txt")); os.IsNotExist(err) {
		t.Fatal("expected untracked file to remain on disk, but it was deleted")
	}
}

func TestRemovePurrFiles_DirectoryRejected(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	testutils.WriteTestFile(t, repo, "subdir/file.txt", "content")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	if err := purrcommands.AddPurrFiles("subdir/file.txt"); err != nil {
		t.Fatalf("AddPurrFiles failed: %v", err)
	}

	if err := purrcommands.RemovePurrFiles("subdir"); err == nil {
		t.Fatal("expected error removing a directory, got nil")
	}
}

func TestRemovePurrFiles_NormalizesRelativePaths(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	testutils.WriteTestFile(t, repo, "nested/file.txt", "content")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	if err := purrcommands.AddPurrFiles("nested/file.txt"); err != nil {
		t.Fatalf("AddPurrFiles failed: %v", err)
	}

	if err := purrcommands.RemovePurrFiles("./nested/../nested/file.txt"); err != nil {
		t.Fatalf("RemovePurrFiles failed for normalized path: %v", err)
	}

	entries, err := index.ReadIndex(filepath.Join(repo, ".purr", "index"))
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty index after normalized removal, got %d entries", len(entries))
	}
}
