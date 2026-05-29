package purrCommands_test

import (
	"os"
	"path/filepath"
	"testing"

	"Persephone/internal/purrCommands"
	"Persephone/internal/testutils"
)

func TestListFiles_EmptyIndex(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// A freshly initialised repo has an empty index — ListFiles should succeed.
	err := purrCommands.ListFiles(repo, false)
	if err != nil {
		t.Errorf("expected no error for empty index, got: %v", err)
	}
}

// chdirTo changes the working directory and returns a cleanup function.
// The caller must defer the returned function to restore the original directory.
func chdirTo(t *testing.T, dir string) func() {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}
}

func TestListFiles_WithFiles(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	// Create test files
	testutils.WriteTestFile(t, repo, "hello.txt", "hello world\n")
	testutils.WriteTestFile(t, repo, "subdir/nested.txt", "nested content\n")

	// AddPurrFiles uses os.Getwd(), so we must be inside the repo
	restore := chdirTo(t, repo)
	defer restore()

	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	err := purrCommands.ListFiles(repo, false)
	if err != nil {
		t.Errorf("expected no error listing files, got: %v", err)
	}
}

func TestListFiles_DebugMode(t *testing.T) {
	repo := testutils.SetupTestRepo(t)

	testutils.WriteTestFile(t, repo, "debug.txt", "debug content\n")

	restore := chdirTo(t, repo)
	defer restore()

	if err := purrCommands.AddPurrFiles("."); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	err := purrCommands.ListFiles(repo, true)
	if err != nil {
		t.Errorf("expected no error in debug mode, got: %v", err)
	}
}

func TestListFiles_NoIndex(t *testing.T) {
	// Create a repo-like directory but without a valid index file
	dir := t.TempDir()
	purrDir := filepath.Join(dir, ".purr")
	if err := os.MkdirAll(purrDir, 0755); err != nil {
		t.Fatalf("failed to create .purr dir: %v", err)
	}
	// No index file written — ReadIndex should fail

	err := purrCommands.ListFiles(dir, false)
	if err == nil {
		t.Fatal("expected error when index file is missing, got nil")
	}
}
