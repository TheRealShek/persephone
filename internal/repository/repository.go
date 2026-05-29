package repository

import (
	"fmt"
	"os"
	"path/filepath"
)

// Repository serves as the central encapsulating handle for a Persephone repository.
//
// Encapsulation Design & Separation of Concerns:
// To prevent filepath construction logic (e.g., `filepath.Join(root, ".purr", "objects", ...)`) from
// scattering across CLI commands, parsers, and platform wrappers, this struct acts as the Single Source of Truth
// for directory mapping. If the internal layout or storage structure of the VCS changes in the future,
// only the helpers in this package need adjustment.
type Repository struct {
	RootDir string // Absolute or clean relative path to the workspace root
}

// Open validates that a `.purr` directory exists at the given path and returns an active Repository handle.
func Open(path string) (*Repository, error) {
	purrDir := filepath.Join(path, ".purr")
	info, err := os.Stat(purrDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a purr repository (or any of the parent directories): .purr")
	}
	return &Repository{RootDir: path}, nil
}

// PurrDir returns the absolute path to the main repository metadata directory.
func (r *Repository) PurrDir() string {
	return filepath.Join(r.RootDir, ".purr")
}

// ObjectsDir returns the folder containing zlib-compressed content-addressed blobs.
func (r *Repository) ObjectsDir() string {
	return filepath.Join(r.PurrDir(), "objects")
}

// IndexPath returns the staging area binary catalog file path.
func (r *Repository) IndexPath() string {
	return filepath.Join(r.PurrDir(), "index")
}

// HeadPath returns the current branch/commit symbolic pointer file path.
func (r *Repository) HeadPath() string {
	return filepath.Join(r.PurrDir(), "HEAD")
}

// RefPath resolves the filesystem location of a specific reference (e.g. "refs/heads/main").
func (r *Repository) RefPath(ref string) string {
	return filepath.Join(r.PurrDir(), ref)
}

// ObjectPath resolves the exact absolute path to a zlib-compressed object file from its 40-character SHA-1 hash.
func (r *Repository) ObjectPath(hash string) string {
	return filepath.Join(r.ObjectsDir(), hash[:2], hash[2:])
}

// ObjectDir returns the parent 2-character hex directory containing the requested object file.
func (r *Repository) ObjectDir(hash string) string {
	return filepath.Join(r.ObjectsDir(), hash[:2])
}

