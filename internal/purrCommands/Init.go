package purrCommands

import (
	"Persephone/internal/platform"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

// InitPurrDirectories bootstraps a new Persephone repository.
//
// Directory Architecture:
//  - `.purr/objects`: Stored content-addressed blobs, trees, and commit snapshots compressed with zlib.
//  - `.purr/refs/heads`: Stored branch pointer files (each contains the 40-char SHA-1 of the tip commit).
//  - `.purr/logs`: Stored operation logs for eventual reflog capabilities.
func InitPurrDirectories(basePath string) (bool, error) {
	purrDir := filepath.Join(basePath, ".purr")

	isNewRepo := false
	if _, err := os.Stat(purrDir); os.IsNotExist(err) {
		isNewRepo = true
	}

	dirs := []string{
		filepath.Join(purrDir, "objects"),
		filepath.Join(purrDir, "refs", "heads"),
		filepath.Join(purrDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false, err
		}
	}

	// Register OS-specific attributes: hides the `.purr` directory on Windows (no-op on Unix/macOS)
	if err := platform.SetHidden(purrDir); err != nil {
		return false, err
	}

	// Bootstrap index file:
	// A VCS requires a valid index to stage files. To prevent subsequent `ReadIndex` calls
	// from failing or crashing on an empty or missing file, we seed it immediately with a
	// valid 12-byte header: magic "DIRC", format version 2, and 0 initial staged entries.
	indexPath := filepath.Join(purrDir, "index")
	if info, err := os.Stat(indexPath); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		var buf bytes.Buffer
		buf.WriteString("DIRC")
		if err := binary.Write(&buf, binary.BigEndian, uint32(2)); err != nil {
			return false, err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint32(0)); err != nil {
			return false, err
		}

		if err := os.WriteFile(indexPath, buf.Bytes(), 0644); err != nil {
			return false, err
		}
	} else if info.IsDir() {
		return false, fmt.Errorf("cannot create index: %s is a directory", indexPath)
	}

	// Set default active branch:
	// Point HEAD symbolically to the standard modern branch "refs/heads/main".
	headPath := filepath.Join(purrDir, "HEAD")
	if info, err := os.Stat(headPath); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		headContent := "ref: refs/heads/main\n"
		if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
			return false, err
		}
	} else if info.IsDir() {
		return false, fmt.Errorf("cannot create HEAD: %s is a directory", headPath)
	}

	return isNewRepo, nil
}

