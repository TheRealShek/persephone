package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"os"
)

/*
WriteBlobWithSHA creates a Git-style blob object from a file and stores it in .purr/objects.
It reads the file, prepends a "blob <size>\x00" header, computes the SHA-1 hash of the
combined data, and writes the zlib-compressed blob to .purr/objects/xx/yyyy... where xx
is the first 2 characters of the hash and yyyy... is the remaining hash. This matches
Git's object storage format, allowing content-addressable storage where the hash serves
as the unique identifier for the file's contents.
*/
func WriteBlobWithSHA(rootDir string, filePath string) ([20]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filePath, err)
		return [20]byte{}, err
	}

	header := fmt.Sprintf("blob %d\x00", len(content)) // `\x00` -> null

	blob := append([]byte(header), content...)

	hash := sha1.Sum(blob)
	hashStr := fmt.Sprintf("%x", hash) // hex string

	// Compress with zlib
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(blob)
	w.Close()

	// Call helper to store object
	err = StoreObject(rootDir, hashStr, compressed.Bytes())
	if err != nil {
		return [20]byte{}, err
	}

	return hash, nil
}

// ComputeTreeSHA1 computes the SHA-1 of a Git-compatible tree object from a list of TreeEntries.
// This function delegates the actual tree object construction to BuildTreeObject, which handles
// sorting, validation, and formatting. The resulting tree object is hashed and the hex SHA-1 is returned.
// Used during the commit process to represent the directory structure and contents at a point in time.
func ComputeTreeSHA1(entries []*TreeEntries) (string, error) {
	treeObj, err := BuildTreeObject(entries)
	if err != nil {
		return "", err
	}

	sha := sha1.Sum(treeObj)
	return fmt.Sprintf("%x", sha[:]), nil
}
