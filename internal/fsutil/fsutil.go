package fsutil

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ExistsAndIsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, err
	}
	if err != nil {
		return false, fmt.Errorf("stat check failed for %s: %w", path, err)
	}
	return info.IsDir(), nil
}

func WalkAndAddFiles(root string, handleFile func(string) error) error {
	return filepath.WalkDir(root, func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %w", entryPath, err)
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if err := handleFile(entryPath); err != nil {
			log.Printf("error handling file %s: %v", entryPath, err)
			return nil
		}
		return nil
	})
}
