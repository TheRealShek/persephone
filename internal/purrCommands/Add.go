package purrCommands

import (
	"Persephone/internal/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

func AddPurrFiles(arg ...string) error {
	// Case: if `purr init` was not done before, gives an error
	targetDir := filepath.Join(".", ".purr")
	ok, err := utils.ExistsAndIsDirectory(targetDir)
	if err != nil || !ok {
		fmt.Printf("Error: .purr directory not initialized — %v\n", err)
		os.Exit(1)
	}
	// Get Current Working Directory
	dirPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("Unable to get Working Directory %s \n", err)
		return nil
	}

	// Case: only `purr add` was written
	if len(arg) == 0 {
		fmt.Println("No Files added")
		return nil
	}

	//Detect if the user passed . (all files) or specific files.
	if len(arg) == 1 && arg[0] == "." {
		addAllPurrFiles(dirPath)
	} else {
		addSpecificPurrFiles(dirPath, arg)
	}
	return nil
}

// Called by func AddPurrFiles() when User passed `purr add .` (all files)
// This function recursively stages all non-hidden files in the working directory for commit.
// It uses goroutines for concurrent file processing, with a worker pool to limit concurrency
// based on CPU cores. Only new or modified files (detected via modification time) are updated
// in the index. Hidden files and directories (starting with '.') are automatically skipped.
func addAllPurrFiles(path string) error {
	// Load all index entries from .purr/index file to IndexEntries
	IndexEntries, _ := utils.ReadIndex(filepath.Join(path, ".purr", "index"))

	// Create a map for faster lookups (path -> entry)
	indexMap := make(map[string]*utils.IndexEntry)
	for i := range IndexEntries {
		indexMap[IndexEntries[i].Path] = &IndexEntries[i]
	}

	// Use up to 5× CPU cores as worker limit
	numWorkers := runtime.NumCPU() * 5
	semaphore := make(chan struct{}, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	utils.WalkAndAddFiles(path, func(filePath string) error {
		wg.Add(1)
		go func(tempPath string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire slot (blocks if at limit)
			defer func() { <-semaphore }() // Release slot when done

			// Getting file Info
			fileInfo, err := os.Stat(tempPath)
			if err != nil {
				log.Printf("failed to stat %s: %v", tempPath, err)
				return
			}

			// Get relative path from repo root
			relPath, err := filepath.Rel(path, tempPath)
			if err != nil {
				log.Printf("failed to get relative path for %s: %v", tempPath, err)
				return
			}

			// Check if file exists in index
			existingEntry, exists := indexMap[relPath]
			if exists {
				if fileInfo.ModTime().Equal(existingEntry.Mtime) {
					return
				}
			}

			// File is new or modified - write blob
			hash, err := utils.WriteBlobWithSHA(path, tempPath)
			if err != nil {
				log.Printf("failed to write blob with SHA for %s: %v", tempPath, err)
				return
			}

			// Create new entry with all fields populated
			newEntry := utils.PopulateAllIndexField(fileInfo, relPath)
			newEntry.Sha1 = hash

			// Update map with lock
			mu.Lock()
			indexMap[relPath] = &newEntry
			mu.Unlock()

		}(filePath)
		return nil
	})

	// Wait for all goroutines to finish
	wg.Wait()

	// Convert map to slice after all updates are complete
	var updatedEntries []utils.IndexEntry
	for _, entry := range indexMap {
		updatedEntries = append(updatedEntries, *entry)
	}

	// Sort entries by path for deterministic output
	sort.Slice(updatedEntries, func(i, j int) bool {
		return updatedEntries[i].Path < updatedEntries[j].Path
	})

	// Write updated index to disk
	indexPath := filepath.Join(path, ".purr", "index")
	if err := utils.WriteIndex(indexPath, updatedEntries); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// Called by func AddPurrFiles() when User passed `purr add file1 ...` (specific files)
// This function stages specific files provided by the user for commit. It validates each file,
// checks if they're within the repository bounds, skips hidden files, and only updates the index
// for new or modified files. Uses goroutines for concurrent file processing with a worker pool
// (5× CPU cores). All map accesses are protected by mutex locks to prevent race conditions.
// The final index entries are sorted alphabetically by path for deterministic output.
func addSpecificPurrFiles(path string, files []string) error {
	// Check for empty file list
	if len(files) == 0 {
		fmt.Println("No files specified")
		return nil
	}

	// Load all index entries from .purr/index file
	IndexEntries, err := utils.ReadIndex(filepath.Join(path, ".purr", "index"))
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	// Create a map for faster lookups (path -> entry)
	indexMap := make(map[string]*utils.IndexEntry)
	for i := range IndexEntries {
		indexMap[IndexEntries[i].Path] = &IndexEntries[i]
	}

	// Counters for tracking results
	var addedCount, skippedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use up to 5× CPU cores as worker limit
	numWorkers := runtime.NumCPU() * 5
	semaphore := make(chan struct{}, numWorkers)

	// Process each specified file concurrently
	for _, filePath := range files {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire slot (blocks if at limit)
			defer func() { <-semaphore }() // Release slot when done

			// Clean and normalize the path
			cleanPath := filepath.Clean(fp)

			// Skip hidden files/directories (starting with .)
			// Check both the input path and each component of the path
			pathParts := strings.Split(cleanPath, string(filepath.Separator))
			isHidden := false
			for _, part := range pathParts {
				if len(part) > 0 && part[0] == '.' {
					isHidden = true
					break
				}
			}
			if isHidden {
				mu.Lock()
				fmt.Printf("Skipping hidden file: %s\n", cleanPath)
				skippedCount++
				mu.Unlock()
				return
			}

			// Convert to absolute path if relative
			absPath := cleanPath
			if !filepath.IsAbs(cleanPath) {
				absPath = filepath.Join(path, cleanPath)
			}

			// Check if file exists
			fileInfo, err := os.Stat(absPath)
			if err != nil {
				mu.Lock()
				fmt.Printf("Error: cannot stat '%s': %v\n", fp, err)
				mu.Unlock()
				return
			}

			// Skip directories
			if fileInfo.IsDir() {
				mu.Lock()
				fmt.Printf("Error: '%s' is a directory (use 'purr add .' to add all files)\n", fp)
				mu.Unlock()
				return
			}

			// Get relative path from repo root
			relPath, err := filepath.Rel(path, absPath)
			if err != nil {
				mu.Lock()
				fmt.Printf("Error: failed to get relative path for '%s': %v\n", fp, err)
				mu.Unlock()
				return
			}

			// Validate file is within repository (not outside with ../)
			if strings.HasPrefix(relPath, "..") {
				mu.Lock()
				fmt.Printf("Error: '%s' is outside repository\n", fp)
				mu.Unlock()
				return
			}

			// Check if file exists in index (with lock protection)
			mu.Lock()
			existingEntry, exists := indexMap[relPath]
			shouldSkip := false
			if exists {
				// Skip if file hasn't been modified
				if fileInfo.ModTime().Equal(existingEntry.Mtime) {
					fmt.Printf("Unchanged: %s\n", relPath)
					skippedCount++
					shouldSkip = true
				}
			}
			mu.Unlock()

			if shouldSkip {
				return
			}

			// File is new or modified - write blob
			hash, err := utils.WriteBlobWithSHA(path, absPath)
			if err != nil {
				mu.Lock()
				fmt.Printf("Error: failed to create blob for '%s': %v\n", fp, err)
				mu.Unlock()
				return
			}

			// Create new entry with all fields populated
			newEntry := utils.PopulateAllIndexField(fileInfo, relPath)
			newEntry.Sha1 = hash

			// Update map with lock
			mu.Lock()
			indexMap[relPath] = &newEntry
			fmt.Printf("Added: %s\n", relPath)
			addedCount++
			mu.Unlock()

		}(filePath)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Only write index if something was added or modified
	if addedCount > 0 {
		// Convert map to slice after all updates are complete
		var updatedEntries []utils.IndexEntry
		for _, entry := range indexMap {
			updatedEntries = append(updatedEntries, *entry)
		}

		// Sort entries by path for deterministic output
		sort.Slice(updatedEntries, func(i, j int) bool {
			return updatedEntries[i].Path < updatedEntries[j].Path
		})

		// Write updated index to disk
		indexPath := filepath.Join(path, ".purr", "index")
		if err := utils.WriteIndex(indexPath, updatedEntries); err != nil {
			return fmt.Errorf("failed to write index: %w", err)
		}

		fmt.Printf("\nSuccessfully added %d file(s) to index", addedCount)
		if skippedCount > 0 {
			fmt.Printf(" (%d skipped)", skippedCount)
		}
		fmt.Println()
	} else {
		fmt.Println("\nNo files were added to index")
	}

	return nil
}
