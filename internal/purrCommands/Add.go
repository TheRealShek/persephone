package purrCommands

import (
	"Persephone/internal/ui"
	"Persephone/internal/utils"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

// AddPurrFiles is the primary controller for staging files.
// It verifies repository integrity (checking that `.purr` has been initialized) and
// routes execution to bulk folder staging (`addAllPurrFiles`) or explicit item staging (`addSpecificPurrFiles`).
func AddPurrFiles(arg ...string) error {
	targetDir := filepath.Join(".", ".purr")
	ok, err := utils.ExistsAndIsDirectory(targetDir)
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
		fmt.Println(ui.Warningf("No files selected to add"))
		return nil
	}

	if len(arg) == 1 && arg[0] == "." {
		return addAllPurrFiles(dirPath)
	} else {
		return addSpecificPurrFiles(dirPath, arg)
	}
}

// addAllPurrFiles recursively crawls the repository root and stages all tracked files.
//
// Concurrency-First Design:
//  1. Worker Pool Bound: We process each file in a separate goroutine. To prevent OS-level
//     exhaustion of file descriptors or stack allocations in massive repositories, we bound active
//     concurrency using a buffered semaphore channel (`semaphore`), capped at `runtime.NumCPU() * 5`.
//  2. Why 5x CPU Cores: File hashing and zlib serialization are heavily disk I/O bound rather than
//     purely CPU bound. Multiplexing multiple workers per logical core ensures that while some workers
//     block on disk read/write, others compute SHA-1 hashes or process network system buffers, maximizing throughput.
//  3. Thread-Safe Critical Sections: A single `sync.Mutex` protects access to the shared `indexMap` and
//     the slice of `processingErrs`. We keep critical sections tightly focused: only locking during map lookups,
//     map updates, and error logging, allowing file system reads and hash calculations to run in parallel.
//  4. Performance Bypass (Stat Caching): Before reading or hashing a file, we check if it is already in the index.
//     If the file's modification time matches the cached `Mtime` in the index, we skip the file entirely.
//     This turns staging into a near-instant O(N) stat scan for unchanged working trees.
//  5. Determinism: Because concurrent goroutines complete in a non-deterministic order, we convert the map
//     to a slice and sort it lexicographically by path before writing to disk. This ensures index updates
//     generate byte-for-byte identical binaries across executions.
func addAllPurrFiles(path string) error {
	IndexEntries, err := utils.ReadIndex(filepath.Join(path, ".purr", "index"))
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	indexMap := make(map[string]*utils.IndexEntry)
	for i := range IndexEntries {
		indexMap[IndexEntries[i].Path] = &IndexEntries[i]
	}

	numWorkers := runtime.NumCPU() * 5
	semaphore := make(chan struct{}, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var processingErrs []error
	var addedCount, skippedCount int
	walkedMap := make(map[string]bool)

	utils.WalkAndAddFiles(path, func(filePath string) error {
		wg.Add(1)
		go func(tempPath string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire token (blocks if active workers >= limit)
			defer func() { <-semaphore }() // Release token back to the bucket

			fileInfo, err := os.Stat(tempPath)
			if err != nil {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("failed to stat %s: %w", tempPath, err))
				mu.Unlock()
				return
			}

			relPath, err := filepath.Rel(path, tempPath)
			if err != nil {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("failed to get relative path for %s: %w", tempPath, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			walkedMap[relPath] = true
			mu.Unlock()

			// Stat cache comparison: skip hashing if Mtime is unchanged
			mu.Lock()
			existingEntry, exists := indexMap[relPath]
			mu.Unlock()
			if exists {
				if fileInfo.ModTime().Equal(existingEntry.Mtime) &&
					uint32(fileInfo.Size()) == existingEntry.Size {
					mu.Lock()
					skippedCount++
					mu.Unlock()
					return
				}
			}

			// Perform expensive I/O operations outside the lock context
			hash, err := utils.WriteBlobWithSHA(path, tempPath)
			if err != nil {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("failed to write blob for %s: %w", tempPath, err))
				mu.Unlock()
				return
			}

			newEntry := utils.PopulateAllIndexField(fileInfo, relPath)
			newEntry.Sha1 = hash

			// Mutex protected updates on the shared map
			mu.Lock()
			indexMap[relPath] = &newEntry
			addedCount++
			mu.Unlock()

		}(filePath)
		return nil
	})

	wg.Wait()

	if len(processingErrs) > 0 {
		return fmt.Errorf("purr add failed: %w", errors.Join(processingErrs...))
	}

	for key := range indexMap {
		if !walkedMap[key] {
			delete(indexMap, key)
		}
	}

	var updatedEntries []utils.IndexEntry
	for _, entry := range indexMap {
		updatedEntries = append(updatedEntries, *entry)
	}

	// Lexicographical sorting is a VCS format invariant for fast lookup performance in standard Git
	slices.SortFunc(updatedEntries, func(a, b utils.IndexEntry) int {
		return strings.Compare(a.Path, b.Path)
	})

	indexPath := filepath.Join(path, ".purr", "index")
	if err := utils.WriteIndex(indexPath, updatedEntries); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	if addedCount > 0 {
		fmt.Printf("%s\n", ui.Successf("Successfully added %d file(s) to index", addedCount))
	} else {
		fmt.Println(ui.Metadata("No files were added to index"))
	}

	return nil
}

// addSpecificPurrFiles stages only the explicit paths supplied as CLI arguments.
// It shares the concurrent hashing worker pool, stat-bypass, and serialization logic of bulk adding,
// but layers path validation checks to ensure:
//  1. Out-of-bounds protection: Blocks staging files residing outside the repository (containing "../").
//  2. Directory filtering: Blocks staging folders directly (routing users to `purr add .` instead).
//  3. Hidden file guards: Skip files starting with `.`, warning the developer to prevent accidental config commits.
func addSpecificPurrFiles(path string, files []string) error {
	if len(files) == 0 {
		fmt.Println("No files specified")
		return nil
	}

	IndexEntries, err := utils.ReadIndex(filepath.Join(path, ".purr", "index"))
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	indexMap := make(map[string]*utils.IndexEntry)
	for i := range IndexEntries {
		indexMap[IndexEntries[i].Path] = &IndexEntries[i]
	}

	var addedCount, skippedCount, removedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup
	var processingErrs []error

	numWorkers := runtime.NumCPU() * 5
	semaphore := make(chan struct{}, numWorkers)

	for _, filePath := range files {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			cleanPath := filepath.Clean(fp)

			// Robust check for hidden paths (e.g. "path/to/.hidden_file" or ".env")
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
				fmt.Printf("%s %s\n", ui.Metadata("Skipping hidden file:"), ui.Metadata(cleanPath))
				skippedCount++
				mu.Unlock()
				return
			}

			absPath := cleanPath
			if !filepath.IsAbs(cleanPath) {
				absPath = filepath.Join(path, cleanPath)
			}

			fileInfo, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					relPath, relErr := filepath.Rel(path, absPath)
					if relErr == nil {
						mu.Lock()
						if _, exists := indexMap[relPath]; exists {
							delete(indexMap, relPath)
							fmt.Printf("removed %s from index\n", relPath)
							removedCount++
						}
						mu.Unlock()
					}
					return
				}
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("cannot stat '%s': %w", fp, err))
				mu.Unlock()
				return
			}

			if fileInfo.IsDir() {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("'%s' is a directory (use 'purr add .' to add all files)", fp))
				mu.Unlock()
				return
			}

			relPath, err := filepath.Rel(path, absPath)
			if err != nil {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("failed to get relative path for '%s': %w", fp, err))
				mu.Unlock()
				return
			}

			// Security check: block path traversal attempts to stage files outside the repo root
			if strings.HasPrefix(relPath, "..") {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("'%s' is outside repository", fp))
				mu.Unlock()
				return
			}

			mu.Lock()
			existingEntry, exists := indexMap[relPath]
			shouldSkip := false
			if exists {
				if fileInfo.ModTime().Equal(existingEntry.Mtime) &&
					uint32(fileInfo.Size()) == existingEntry.Size {
					fmt.Printf("%s %s\n", ui.Modified("Unchanged:"), ui.StyledPath(relPath))
					skippedCount++
					shouldSkip = true
				}
			}
			mu.Unlock()

			if shouldSkip {
				return
			}

			hash, err := utils.WriteBlobWithSHA(path, absPath)
			if err != nil {
				mu.Lock()
				processingErrs = append(processingErrs, fmt.Errorf("failed to create blob for '%s': %w", fp, err))
				mu.Unlock()
				return
			}

			newEntry := utils.PopulateAllIndexField(fileInfo, relPath)
			newEntry.Sha1 = hash

			mu.Lock()
			indexMap[relPath] = &newEntry
			fmt.Printf("%s %s\n", ui.Added("Added:"), ui.StyledPath(relPath))
			addedCount++
			mu.Unlock()

		}(filePath)
	}

	wg.Wait()

	if len(processingErrs) > 0 {
		return fmt.Errorf("purr add failed: %w", errors.Join(processingErrs...))
	}

	if addedCount > 0 || removedCount > 0 {
		var updatedEntries []utils.IndexEntry
		for _, entry := range indexMap {
			updatedEntries = append(updatedEntries, *entry)
		}

		slices.SortFunc(updatedEntries, func(a, b utils.IndexEntry) int {
			return strings.Compare(a.Path, b.Path)
		})

		indexPath := filepath.Join(path, ".purr", "index")
		if err := utils.WriteIndex(indexPath, updatedEntries); err != nil {
			return fmt.Errorf("failed to write index: %w", err)
		}

		if addedCount > 0 {
			fmt.Printf("\n%s", ui.Successf("Successfully added %d file(s) to index", addedCount))
		}
		if skippedCount > 0 {
			fmt.Printf(" %s", ui.Metadataf("(%d skipped)", skippedCount))
		}
		if addedCount > 0 || skippedCount > 0 {
			fmt.Println()
		}
	} else {
		fmt.Println()
		fmt.Println(ui.Metadata("No files were added to index"))
	}

	return nil
}
