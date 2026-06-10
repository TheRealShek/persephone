package purrCommands

import (
	"Persephone/internal/index"
	"Persephone/internal/ui"

	"fmt"
	"os"
	"path/filepath"
)

// ListFiles reads `.purr/index` and formats the staged files list.
//
// Dual Presentation Models:
//   - Standard Mode (Default): Focuses on clean file lists, showing paths, short 7-char SHA-1 prefixes,
//     and standard octal file permissions. This is optimized for rapid developer scanning.
//   - Debug Mode (`showDebug = true`): Dumps the low-level stat cache metadata of each staged index record
//     (inodes, device IDs, timestamps, conflict stages, etc.). This acts as a vital tool for VCS maintainers
//     verifying binary structure alignment, filesystem change-detection, and stat caching correctness.
func ListFiles(rootDir string, showDebug bool) error {
	purrDir := filepath.Join(rootDir, ".purr")
	if _, err := os.Stat(purrDir); os.IsNotExist(err) {
		return fmt.Errorf("not a purr repository")
	} else if err != nil {
		return err
	}

	indexPath := filepath.Join(purrDir, "index")
	entries, err := index.ReadIndex(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println(ui.Metadata("No files staged"))
		return nil
	}

	plural := "s"
	if len(entries) == 1 {
		plural = ""
	}
	headerText := fmt.Sprintf("%d file%s staged:", len(entries), plural)

	if showDebug {
		fmt.Printf("%s\n\n", ui.SectionHeader(headerText))
		for i, entry := range entries {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("%s %s\n", ui.Metadata("Path:"), ui.StyledPath(entry.Path))
			fmt.Printf("%s %s\n", ui.Metadata("SHA1:"), ui.Metadata(fmt.Sprintf("%x", entry.Sha1)))
			fmt.Printf("%s %s\n", ui.Metadata("Mode:"), ui.Metadata(fmt.Sprintf("%06o", entry.Mode)))
			fmt.Printf("%s %s\n", ui.Metadata("Size:"), ui.Metadata(fmt.Sprintf("%d bytes", entry.Size)))
			fmt.Printf("%s %s\n", ui.Metadata("Mtime:"), ui.Metadata(entry.Mtime.Format("2006-01-02 15:04:05")))
			fmt.Printf("%s %s\n", ui.Metadata("Ctime:"), ui.Metadata(entry.Ctime.Format("2006-01-02 15:04:05")))
			fmt.Printf("%s %s\n", ui.Metadata("Dev:"), ui.Metadata(fmt.Sprintf("%d", entry.Dev)))
			fmt.Printf("%s %s\n", ui.Metadata("Ino:"), ui.Metadata(fmt.Sprintf("%d", entry.Ino)))
			fmt.Printf("%s %s\n", ui.Metadata("UID:"), ui.Metadata(fmt.Sprintf("%d", entry.Uid)))
			fmt.Printf("%s %s\n", ui.Metadata("GID:"), ui.Metadata(fmt.Sprintf("%d", entry.Gid)))
			fmt.Printf("%s %s\n", ui.Metadata("Stage:"), ui.Metadata(fmt.Sprintf("%d", entry.Stage)))
		}
	} else {
		fmt.Printf("%s\n\n", ui.SectionHeader(headerText))

		fmt.Println(ui.LsHeader())
		for _, entry := range entries {
			shortSha := fmt.Sprintf("%x", entry.Sha1)[:7]
			fmt.Println(ui.LsRow(entry.Path, shortSha, fmt.Sprintf("%06o", entry.Mode)))
		}
	}

	return nil
}
