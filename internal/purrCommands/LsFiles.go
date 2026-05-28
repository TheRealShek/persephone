package purrCommands

import (
	"Persephone/internal/utils"
	"fmt"
	"path/filepath"
)

// ListFiles reads the index and displays file information
func ListFiles(rootDir string, showDebug bool) error {
	indexPath := filepath.Join(rootDir, ".purr", "index")
	entries, err := utils.ReadIndex(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No files in index")
		return nil
	}

	if showDebug {
		// Detailed output similar to git ls-files --debug
		fmt.Printf("Found %d file(s) in index:\n\n", len(entries))
		for i, entry := range entries {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("  Path: %s\n", entry.Path)
			fmt.Printf("  SHA1: %x\n", entry.Sha1)
			fmt.Printf("  Mode: %06o\n", entry.Mode)
			fmt.Printf("  Size: %d bytes\n", entry.Size)
			fmt.Printf("  Mtime: %s\n", entry.Mtime.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Ctime: %s\n", entry.Ctime.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Dev: %d\n", entry.Dev)
			fmt.Printf("  Ino: %d\n", entry.Ino)
			fmt.Printf("  UID: %d\n", entry.Uid)
			fmt.Printf("  GID: %d\n", entry.Gid)
			fmt.Printf("  Stage: %d\n", entry.Stage)
		}
	} else {
		// Simple output (default)
		fmt.Printf("Found %d file(s) in index:\n\n", len(entries))
		for _, entry := range entries {
			fmt.Printf("%x %06o %s\n", entry.Sha1, entry.Mode, entry.Path)
		}
	}

	return nil
}
