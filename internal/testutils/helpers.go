package testutils

import (
	"Persephone/internal/purrCommands"
	"os"
	"path/filepath"
	"testing"
)

// SetupTestRepo creates a temporary directory, initializes it as a purr repo,
// sets up a mock config file, and returns the path to the repo.
func SetupTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	err := purrCommands.InitPurrDirectories(repo)
	if err != nil {
		t.Fatalf("failed to init purr repo: %v", err)
	}

	configPath := filepath.Join(repo, ".purrconfig")
	configContent := `{"user_name":"Test User","user_email":"test@example.com"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	t.Setenv("PURR_CONFIG_PATH", configPath)

	return repo
}

// WriteTestFile writes a file with the given content in the specified directory.
func WriteTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directories for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
	return path
}
