package objects

import (
	"os"
	"path/filepath"
	"sync"
)

var createdDirs sync.Map

func StoreObject(rootDir string, hashStr string, data []byte) error {
	objectPath := filepath.Join(rootDir, ".purr", "objects", hashStr[:2], hashStr[2:])
	if _, err := os.Stat(objectPath); err == nil {
		return nil
	}
	dir := filepath.Dir(objectPath)
	val, _ := createdDirs.LoadOrStore(dir, &sync.Once{})
	var mkdirErr error
	val.(*sync.Once).Do(func() {
		mkdirErr = os.MkdirAll(dir, 0755)
	})
	if mkdirErr != nil {
		return mkdirErr
	}
	return os.WriteFile(objectPath, data, 0644)
}
