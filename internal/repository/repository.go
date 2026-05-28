package repository

import (
	"fmt"
	"os"
	"path/filepath"
)

// Repository is the central handle for a purr repository.
// It holds the root directory and provides path resolution helpers
// so that no other package needs to construct .purr/ paths directly.
type Repository struct {
	RootDir string
}

// Open returns a Repository handle for an existing .purr repository at path.
func Open(path string) (*Repository, error) {
	purrDir := filepath.Join(path, ".purr")
	info, err := os.Stat(purrDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a purr repository (or any of the parent directories): .purr")
	}
	return &Repository{RootDir: path}, nil
}

// PurrDir returns the path to the .purr directory.
func (r *Repository) PurrDir() string {
	return filepath.Join(r.RootDir, ".purr")
}

// ObjectsDir returns the path to the objects directory.
func (r *Repository) ObjectsDir() string {
	return filepath.Join(r.PurrDir(), "objects")
}

// IndexPath returns the path to the index file.
func (r *Repository) IndexPath() string {
	return filepath.Join(r.PurrDir(), "index")
}

// HeadPath returns the path to the HEAD file.
func (r *Repository) HeadPath() string {
	return filepath.Join(r.PurrDir(), "HEAD")
}

// RefPath returns the path to a reference file within .purr.
func (r *Repository) RefPath(ref string) string {
	return filepath.Join(r.PurrDir(), ref)
}

// ObjectPath returns the path where an object with the given hash is stored.
func (r *Repository) ObjectPath(hash string) string {
	return filepath.Join(r.ObjectsDir(), hash[:2], hash[2:])
}

// ObjectDir returns the directory for an object hash prefix (first 2 chars).
func (r *Repository) ObjectDir(hash string) string {
	return filepath.Join(r.ObjectsDir(), hash[:2])
}
