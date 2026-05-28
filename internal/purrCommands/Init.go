package purrCommands

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

func InitPurrDirectories(basePath string) error {
	purrDir := filepath.Join(basePath, ".purr")

	dirs := []string{
		filepath.Join(purrDir, "objects"),
		filepath.Join(purrDir, "refs", "heads"),
		filepath.Join(purrDir, "logs"),
	}

	// Create all directories
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Set hidden attribute on Windows (only if not already hidden)
	if runtime.GOOS == "windows" {
		purrDirPtr, err := syscall.UTF16PtrFromString(purrDir)
		if err != nil {
			return err
		}

		attrs, err := syscall.GetFileAttributes(purrDirPtr)
		if err != nil {
			return err
		}

		// Only set if not already hidden
		if attrs&syscall.FILE_ATTRIBUTE_HIDDEN == 0 {
			err = syscall.SetFileAttributes(purrDirPtr, attrs|syscall.FILE_ATTRIBUTE_HIDDEN)
			if err != nil {
				return err
			}
		}
	}

	// Create index file with valid header if it doesn't exist
	indexPath := filepath.Join(purrDir, "index")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Write valid 12-byte header: "DIRC" + version 2 + 0 entries
		var buf bytes.Buffer
		buf.WriteString("DIRC")                         // Magic (4 bytes)
		binary.Write(&buf, binary.BigEndian, uint32(2)) // Version (4 bytes)
		binary.Write(&buf, binary.BigEndian, uint32(0)) // Entry count (4 bytes)

		if err := os.WriteFile(indexPath, buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	// Create HEAD file if it doesn't exist
	headPath := filepath.Join(purrDir, "HEAD")
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		// Point to refs/heads/main by default
		headContent := "ref: refs/heads/main\n"
		if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
			return err
		}
	}

	return nil
}
