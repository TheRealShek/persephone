package purrcommands

import (
	"persephone/internal/platform"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrRepositoryAlreadyInitialized = errors.New("repository already initialized")

// InitPurrDirectories bootstraps a new Persephone repository.
//
// Directory Architecture:
//   - `.purr/objects`: Stored content-addressed blobs, trees, and commit snapshots compressed with zlib.
//   - `.purr/refs/heads`: Stored branch pointer files (each contains the 40-char SHA-1 of the tip commit).
//   - `.purr/logs`: Stored operation logs for eventual reflog capabilities.
//
// Initialization is deliberately create-only. If `.purr` already exists, callers receive a sentinel
// error before any metadata is touched. The CLI uses that signal to request explicit confirmation
// before calling ReinitializePurrDirectories.
func InitPurrDirectories(basePath string) error {
	purrDir := filepath.Join(basePath, ".purr")
	if info, err := os.Stat(purrDir); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%w: %s already exists", ErrRepositoryAlreadyInitialized, purrDir)
		}
		return fmt.Errorf("cannot initialize repository: %s exists and is not a directory", purrDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect repository metadata path: %w", err)
	}

	return ensurePurrDirectories(purrDir)
}

// ReinitializePurrDirectories restores missing repository scaffolding after the caller has received
// explicit user confirmation. Existing index, HEAD, refs, and objects are preserved because rerunning
// initialization is a repair operation, not a repository reset.
func ReinitializePurrDirectories(basePath string) error {
	purrDir := filepath.Join(basePath, ".purr")
	info, err := os.Stat(purrDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cannot reinitialize repository: %s does not exist", purrDir)
		}
		return fmt.Errorf("cannot inspect repository metadata path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot reinitialize repository: %s is not a directory", purrDir)
	}

	return ensurePurrDirectories(purrDir)
}

func ensurePurrDirectories(purrDir string) error {
	dirs := []string{
		filepath.Join(purrDir, "objects"),
		filepath.Join(purrDir, "refs", "heads"),
		filepath.Join(purrDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Register OS-specific attributes: hides the `.purr` directory on Windows (no-op on Unix/macOS)
	if err := platform.SetHidden(purrDir); err != nil {
		return err
	}

	// Bootstrap index file:
	// A VCS requires a valid index to stage files. To prevent subsequent `ReadIndex` calls
	// from failing or crashing on an empty or missing file, we seed it immediately with a
	// valid 12-byte header: magic "DIRC", format version 2, and 0 initial staged entries.
	indexPath := filepath.Join(purrDir, "index")
	if info, err := os.Stat(indexPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		var buf bytes.Buffer
		buf.WriteString("DIRC")
		if err := binary.Write(&buf, binary.BigEndian, uint32(2)); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint32(0)); err != nil {
			return err
		}

		if err := os.WriteFile(indexPath, buf.Bytes(), 0644); err != nil {
			return err
		}
	} else if info.IsDir() {
		return fmt.Errorf("cannot create index: %s is a directory", indexPath)
	}

	// Set default active branch:
	// Point HEAD symbolically to the standard modern branch "refs/heads/main".
	headPath := filepath.Join(purrDir, "HEAD")
	if info, err := os.Stat(headPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		headContent := "ref: refs/heads/main\n"
		if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
			return err
		}
	} else if info.IsDir() {
		return fmt.Errorf("cannot create HEAD: %s is a directory", headPath)
	}

	return nil
}
