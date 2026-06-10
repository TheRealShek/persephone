package purrcommands

import (
	"persephone/internal/fsutil"
	"persephone/internal/index"
	"persephone/internal/ui"

	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RemovePurrFiles removes tracked files from the index and deletes them from the working tree.
// The command treats paths as repo-relative so removals stay scoped to the current repository root.
func RemovePurrFiles(arg ...string) error {
	targetDir := filepath.Join(".", ".purr")
	ok, err := fsutil.ExistsAndIsDirectory(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check repository: %w", err)
	}
	if !ok {
		return fmt.Errorf(".purr directory not initialized")
	}

	dirPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get working directory: %w", err)
	}

	if len(arg) == 0 {
		fmt.Println(ui.Warningf("No files selected to remove"))
		return nil
	}

	indexPath := filepath.Join(dirPath, ".purr", "index")
	entries, err := index.ReadIndex(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	indexMap := make(map[string]*index.IndexEntry)
	for i := range entries {
		indexMap[entries[i].Path] = &entries[i]
	}

	removedCount := 0

	// Pass 1: Validate all paths
	for _, filePath := range arg {
		cleanPath := filepath.Clean(filePath)
		absPath := cleanPath
		if !filepath.IsAbs(cleanPath) {
			absPath = filepath.Join(dirPath, cleanPath)
		}

		relPath, err := filepath.Rel(dirPath, absPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for '%s': %w", filePath, err)
		}
		relPath = filepath.Clean(relPath)
		if strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("'%s' is outside repository", filePath)
		}

		if _, exists := indexMap[relPath]; !exists {
			return fmt.Errorf("'%s' is not tracked", filePath)
		}

		if info, err := os.Stat(absPath); err == nil {
			if info.IsDir() {
				return fmt.Errorf("'%s' is a directory (directory removal is not supported)", filePath)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat '%s': %w", filePath, err)
		}
	}

	// Pass 2: Execute deletions
	for _, filePath := range arg {
		cleanPath := filepath.Clean(filePath)
		absPath := cleanPath
		if !filepath.IsAbs(cleanPath) {
			absPath = filepath.Join(dirPath, cleanPath)
		}

		relPath, _ := filepath.Rel(dirPath, absPath)
		relPath = filepath.Clean(relPath)

		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove '%s': %w", filePath, err)
		}

		delete(indexMap, relPath)
		removedCount++
		fmt.Printf("%s %s\n", ui.Removed("Removed:"), ui.StyledPath(relPath))
	}

	var updatedEntries []index.IndexEntry
	for _, entry := range indexMap {
		updatedEntries = append(updatedEntries, *entry)
	}

	sort.Slice(updatedEntries, func(i, j int) bool {
		return updatedEntries[i].Path < updatedEntries[j].Path
	})

	if err := index.WriteIndex(indexPath, updatedEntries); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	fmt.Printf("%s\n", ui.Successf("Successfully removed %d file(s)", removedCount))
	return nil
}
