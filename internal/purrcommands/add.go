package purrcommands

import (
	"persephone/internal/fsutil"
	"persephone/internal/hash"
	"persephone/internal/index"
	"persephone/internal/ui"

	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// AddPurrFiles is the primary controller for staging files.
// It verifies repository integrity (checking that `.purr` has been initialized) and
// routes execution to bulk folder staging (`addAllPurrFiles`) or explicit item staging (`addSpecificPurrFiles`).
func AddPurrFiles(arg ...string) error {
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
	IndexEntries, err := index.ReadIndex(filepath.Join(path, ".purr", "index"))
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	indexMap := make(map[string]*index.IndexEntry)
	for i := range IndexEntries {
		indexMap[IndexEntries[i].Path] = &IndexEntries[i]
	}

	numWorkers := runtime.NumCPU() * 5
	jobs := make(chan string, numWorkers*4)
	var wg sync.WaitGroup
	var mu sync.RWMutex
	var walkMu sync.Mutex
	var processingErrs []error
	errCh := make(chan error, numWorkers)
	errDone := make(chan struct{})
	go func() {
		for err := range errCh {
			processingErrs = append(processingErrs, err)
		}
		close(errDone)
	}()
	var addedCount, skippedCount atomic.Int32
	walkedMap := make(map[string]bool)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tempPath := range jobs {
				fileInfo, err := os.Stat(tempPath)
				if err != nil {
					errCh <- fmt.Errorf("failed to stat %s: %w", tempPath, err)
					continue
				}

				relPath, err := filepath.Rel(path, tempPath)
				if err != nil {
					errCh <- fmt.Errorf("failed to get relative path for %s: %w", tempPath, err)
					continue
				}

				walkMu.Lock()
				walkedMap[relPath] = true
				walkMu.Unlock()

				// Stat cache comparison: skip hashing if Mtime is unchanged
				mu.RLock()
				existingEntry, exists := indexMap[relPath]
				shouldSkip := false
				if exists {
					if fileInfo.ModTime().Equal(existingEntry.Mtime) &&
						uint32(fileInfo.Size()) == existingEntry.Size {
						shouldSkip = true
					}
				}
				mu.RUnlock()

				if shouldSkip {
					skippedCount.Add(1)
					continue
				}

				// Perform expensive I/O operations outside the lock context
				hash, err := hash.WriteBlobWithSHA(path, tempPath)
				if err != nil {
					errCh <- fmt.Errorf("failed to write blob for %s: %w", tempPath, err)
					continue
				}

				newEntry := index.PopulateAllIndexField(fileInfo, relPath)
				newEntry.Sha1 = hash

				// Mutex protected updates on the shared map
				mu.Lock()
				indexMap[relPath] = &newEntry
				mu.Unlock()
				addedCount.Add(1)
			}
		}()
	}

	var walkErr error
	go func() {
		walkErr = fsutil.WalkAndAddFiles(path, func(filePath string) error {
			jobs <- filePath
			return nil
		})
		close(jobs)
	}()

	wg.Wait()
	close(errCh)
	<-errDone

	if walkErr != nil {
		fmt.Printf("%s\n", ui.Warningf("Directory walk encountered an error: %v", walkErr))
	}

	if len(processingErrs) > 0 {
		for _, err := range processingErrs {
			fmt.Printf("%s\n", ui.Warningf("Worker error: %v", err))
		}
		return fmt.Errorf("purr add completed with %d error(s)", len(processingErrs))
	}

	for key := range indexMap {
		if !walkedMap[key] {
			delete(indexMap, key)
		}
	}

	var updatedEntries []index.IndexEntry
	for _, entry := range indexMap {
		updatedEntries = append(updatedEntries, *entry)
	}

	// Lexicographical sorting is a VCS format invariant for fast lookup performance in standard Git
	slices.SortFunc(updatedEntries, func(a, b index.IndexEntry) int {
		return strings.Compare(a.Path, b.Path)
	})

	indexPath := filepath.Join(path, ".purr", "index")
	if err := index.WriteIndex(indexPath, updatedEntries); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	addedVal := addedCount.Load()
	if addedVal > 0 {
		fmt.Printf("%s\n", ui.Successf("Successfully added %d file(s) to index", addedVal))
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

	IndexEntries, err := index.ReadIndex(filepath.Join(path, ".purr", "index"))
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	indexMap := make(map[string]*index.IndexEntry)
	for i := range IndexEntries {
		indexMap[IndexEntries[i].Path] = &IndexEntries[i]
	}

	var addedCount, skippedCount, removedCount atomic.Int32
	var mu sync.RWMutex
	var printMu sync.Mutex
	var wg sync.WaitGroup
	var processingErrs []error

	numWorkers := runtime.NumCPU() * 5
	errCh := make(chan error, numWorkers)
	errDone := make(chan struct{})
	go func() {
		for err := range errCh {
			processingErrs = append(processingErrs, err)
		}
		close(errDone)
	}()
	jobs := make(chan string, numWorkers*4)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fp := range jobs {
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
					printMu.Lock()
					fmt.Printf("%s %s\n", ui.Metadata("Skipping hidden file:"), ui.Metadata(cleanPath))
					printMu.Unlock()
					skippedCount.Add(1)
					continue
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
							_, exists := indexMap[relPath]
							if exists {
								delete(indexMap, relPath)
							}
							mu.Unlock()
							if exists {
								printMu.Lock()
								fmt.Printf("removed %s from index\n", relPath)
								printMu.Unlock()
								removedCount.Add(1)
							}
						}
						continue
					}
					errCh <- fmt.Errorf("cannot stat '%s': %w", fp, err)
					continue
				}

				if fileInfo.IsDir() {
					errCh <- fmt.Errorf("'%s' is a directory (use 'purr add .' to add all files)", fp)
					continue
				}

				relPath, err := filepath.Rel(path, absPath)
				if err != nil {
					errCh <- fmt.Errorf("failed to get relative path for '%s': %w", fp, err)
					continue
				}

				// Security check: block path traversal attempts to stage files outside the repo root
				if strings.HasPrefix(relPath, "..") {
					errCh <- fmt.Errorf("'%s' is outside repository", fp)
					continue
				}

				mu.RLock()
				existingEntry, exists := indexMap[relPath]
				shouldSkip := false
				if exists {
					if fileInfo.ModTime().Equal(existingEntry.Mtime) &&
						uint32(fileInfo.Size()) == existingEntry.Size {
						shouldSkip = true
					}
				}
				mu.RUnlock()

				if shouldSkip {
					printMu.Lock()
					fmt.Printf("%s %s\n", ui.Modified("Unchanged:"), ui.StyledPath(relPath))
					printMu.Unlock()
					skippedCount.Add(1)
					continue
				}

				hash, err := hash.WriteBlobWithSHA(path, absPath)
				if err != nil {
					errCh <- fmt.Errorf("failed to create blob for '%s': %w", fp, err)
					continue
				}

				newEntry := index.PopulateAllIndexField(fileInfo, relPath)
				newEntry.Sha1 = hash

				mu.Lock()
				indexMap[relPath] = &newEntry
				mu.Unlock()

				printMu.Lock()
				fmt.Printf("%s %s\n", ui.Added("Added:"), ui.StyledPath(relPath))
				printMu.Unlock()
				addedCount.Add(1)
			}
		}()
	}

	for _, filePath := range files {
		jobs <- filePath
	}
	close(jobs)

	wg.Wait()
	close(errCh)
	<-errDone

	if len(processingErrs) > 0 {
		for _, err := range processingErrs {
			fmt.Printf("%s\n", ui.Warningf("Worker error: %v", err))
		}
		return fmt.Errorf("purr add completed with %d error(s)", len(processingErrs))
	}

	addedVal := addedCount.Load()
	skippedVal := skippedCount.Load()
	removedVal := removedCount.Load()

	if addedVal > 0 || removedVal > 0 {
		var updatedEntries []index.IndexEntry
		for _, entry := range indexMap {
			updatedEntries = append(updatedEntries, *entry)
		}

		slices.SortFunc(updatedEntries, func(a, b index.IndexEntry) int {
			return strings.Compare(a.Path, b.Path)
		})

		indexPath := filepath.Join(path, ".purr", "index")
		if err := index.WriteIndex(indexPath, updatedEntries); err != nil {
			return fmt.Errorf("failed to write index: %w", err)
		}

		if addedVal > 0 {
			fmt.Printf("\n%s", ui.Successf("Successfully added %d file(s) to index", addedVal))
		}
		if skippedVal > 0 {
			fmt.Printf(" %s", ui.Metadataf("(%d skipped)", skippedVal))
		}
		if addedVal > 0 || skippedVal > 0 {
			fmt.Println()
		}
	} else {
		fmt.Println()
		fmt.Println(ui.Metadata("No files were added to index"))
	}

	return nil
}
