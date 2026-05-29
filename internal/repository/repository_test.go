package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"Persephone/internal/repository"
)

func TestOpen_ValidRepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".purr"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	repo, err := repository.Open(dir)
	if err != nil {
		t.Fatalf("expected Open to succeed, got: %v", err)
	}
	if repo.RootDir != dir {
		t.Errorf("RootDir = %q, want %q", repo.RootDir, dir)
	}
}

func TestOpen_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	// No .purr directory created

	_, err := repository.Open(dir)
	if err == nil {
		t.Fatal("expected error when .purr directory does not exist, got nil")
	}
}

func TestOpen_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	// Create .purr as a regular file, not a directory
	purrPath := filepath.Join(dir, ".purr")
	if err := os.WriteFile(purrPath, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := repository.Open(dir)
	if err == nil {
		t.Fatal("expected error when .purr is a file, got nil")
	}
}

// newRepo is a helper that creates a Repository with the given root without
// going through Open (avoids needing a .purr directory on disk for path tests).
func newRepo(root string) *repository.Repository {
	return &repository.Repository{RootDir: root}
}

func TestRepository_PurrDir(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	want := filepath.Join(root, ".purr")
	if got := repo.PurrDir(); got != want {
		t.Errorf("PurrDir() = %q, want %q", got, want)
	}
}

func TestRepository_ObjectsDir(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	want := filepath.Join(root, ".purr", "objects")
	if got := repo.ObjectsDir(); got != want {
		t.Errorf("ObjectsDir() = %q, want %q", got, want)
	}
}

func TestRepository_IndexPath(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	want := filepath.Join(root, ".purr", "index")
	if got := repo.IndexPath(); got != want {
		t.Errorf("IndexPath() = %q, want %q", got, want)
	}
}

func TestRepository_HeadPath(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	want := filepath.Join(root, ".purr", "HEAD")
	if got := repo.HeadPath(); got != want {
		t.Errorf("HeadPath() = %q, want %q", got, want)
	}
}

func TestRepository_RefPath(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "simple ref",
			ref:  "refs/heads/main",
			want: filepath.Join(root, ".purr", "refs/heads/main"),
		},
		{
			name: "tag ref",
			ref:  "refs/tags/v1.0",
			want: filepath.Join(root, ".purr", "refs/tags/v1.0"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := repo.RefPath(tc.ref); got != tc.want {
				t.Errorf("RefPath(%q) = %q, want %q", tc.ref, got, tc.want)
			}
		})
	}
}

func TestRepository_ObjectPath(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	hash := "aabbccddee1122334455aabbccddee1122334455"
	want := filepath.Join(root, ".purr", "objects", "aa", "bbccddee1122334455aabbccddee1122334455")
	if got := repo.ObjectPath(hash); got != want {
		t.Errorf("ObjectPath(%q) = %q, want %q", hash, got, want)
	}
}

func TestRepository_ObjectDir(t *testing.T) {
	root := "/fake/repo"
	repo := newRepo(root)

	hash := "aabbccddee1122334455aabbccddee1122334455"
	want := filepath.Join(root, ".purr", "objects", "aa")
	if got := repo.ObjectDir(hash); got != want {
		t.Errorf("ObjectDir(%q) = %q, want %q", hash, got, want)
	}
}
