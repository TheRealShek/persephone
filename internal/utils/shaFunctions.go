package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"os"
)

// WriteBlobWithSHA reads a file, serializes it into a Git-compatible blob object,
// hashes it to determine its content-addressable ID, compresses it, and stores it on disk.
//
// Key VCS Invariants & Design Choices:
//  1. Content-Addressable Hashing: We prepend a header `blob {size}\x00` to the raw file bytes.
//     Including the object type and length in the hashed payload guarantees that the resulting
//     SHA-1 is unique to both the content and its type, preventing collisions with tree/commit objects.
//  2. Memory Trade-Off: The entire file is read into memory at once. While this is simple and highly
//     efficient for small to medium repositories, very large files (GB range) would cause memory spikes.
//     For an experimental VCS, this is an acceptable simplicity-performance trade-off.
//  3. Object Storage Fan-Out: The 40-character hex SHA-1 is split: first 2 characters as a directory name,
//     next 38 as the file name under `.purr/objects/xx/yyyy...`. This "fan-out" structure prevents a single
//     flat directory from containing thousands of files, which degrades directory listing performance
//     on many operating system filesystems (e.g. ext4, NTFS).
func WriteBlobWithSHA(rootDir string, filePath string) ([20]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filePath, err)
		return [20]byte{}, err
	}

	header := fmt.Sprintf("blob %d\x00", len(content))

	blob := append([]byte(header), content...)

	hash := sha1.Sum(blob)
	hashStr := fmt.Sprintf("%x", hash)

	// Compress using standard zlib to match Git-compatible storage and optimize disk utilization
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	if _, err := w.Write(blob); err != nil {
		return [20]byte{}, fmt.Errorf("failed to compress object: %w", err)
	}
	if err := w.Close(); err != nil {
		return [20]byte{}, fmt.Errorf("failed to finalize compression: %w", err)
	}

	// Write compressed object payload to the content-addressable database
	err = StoreObject(rootDir, hashStr, compressed.Bytes())
	if err != nil {
		return [20]byte{}, err
	}

	return hash, nil
}

// ComputeTreeSHA1 generates a Git-compatible tree object from staged files and hashes it.
// It delegates construction to BuildTreeObject to get sorted deterministic binary formatting.
// The tree SHA-1 represents the direct state of directory hierarchy at the time of commit.
func ComputeTreeSHA1(rootDir string, entries []*TreeEntries) (string, error) {
	treeObj, err := BuildTreeObject(rootDir, entries)
	if err != nil {
		return "", err
	}

	sha := sha1.Sum(treeObj)
	return fmt.Sprintf("%x", sha[:]), nil
}

