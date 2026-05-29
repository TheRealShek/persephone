package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupPurrObjectsDir creates a minimal .purr/objects directory structure
// inside the given root, suitable for WriteBlobWithSHA tests.
func setupPurrObjectsDir(t *testing.T, root string) {
	t.Helper()
	objDir := filepath.Join(root, ".purr", "objects")
	if err := os.MkdirAll(objDir, 0755); err != nil {
		t.Fatalf("failed to create .purr/objects: %v", err)
	}
}

// writeTestFile creates a file with the given content inside dir.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
	return path
}

func TestWriteBlobWithSHA_CreatesObject(t *testing.T) {
	root := t.TempDir()
	setupPurrObjectsDir(t, root)

	filePath := writeTestFile(t, root, "test.txt", "some content\n")

	hash, err := WriteBlobWithSHA(root, filePath)
	if err != nil {
		t.Fatalf("WriteBlobWithSHA() unexpected error: %v", err)
	}

	hashStr := fmt.Sprintf("%x", hash)
	objPath := filepath.Join(root, ".purr", "objects", hashStr[:2], hashStr[2:])

	info, err := os.Stat(objPath)
	if err != nil {
		t.Fatalf("expected object file at %s, got error: %v", objPath, err)
	}
	if info.Size() == 0 {
		t.Errorf("object file is empty, expected compressed blob data")
	}
}

func TestWriteBlobWithSHA_CorrectHash(t *testing.T) {
	root := t.TempDir()
	setupPurrObjectsDir(t, root)

	content := "hello\n"
	filePath := writeTestFile(t, root, "hello.txt", content)

	// Compute expected hash: SHA-1 of "blob 6\0hello\n"
	header := fmt.Sprintf("blob %d\x00", len(content))
	blob := append([]byte(header), []byte(content)...)
	expectedHash := sha1.Sum(blob)

	gotHash, err := WriteBlobWithSHA(root, filePath)
	if err != nil {
		t.Fatalf("WriteBlobWithSHA() unexpected error: %v", err)
	}

	if gotHash != expectedHash {
		t.Errorf("hash mismatch:\n  got:  %x\n  want: %x", gotHash, expectedHash)
	}
}

func TestWriteBlobWithSHA_CompressedContent(t *testing.T) {
	root := t.TempDir()
	setupPurrObjectsDir(t, root)

	content := "decompression test data\n"
	filePath := writeTestFile(t, root, "data.txt", content)

	hash, err := WriteBlobWithSHA(root, filePath)
	if err != nil {
		t.Fatalf("WriteBlobWithSHA() unexpected error: %v", err)
	}

	// Read back the stored object
	hashStr := fmt.Sprintf("%x", hash)
	objPath := filepath.Join(root, ".purr", "objects", hashStr[:2], hashStr[2:])
	compressed, err := os.ReadFile(objPath)
	if err != nil {
		t.Fatalf("failed to read stored object: %v", err)
	}

	// Decompress
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("failed to create zlib reader: %v", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to decompress object: %v", err)
	}

	// Verify the decompressed content matches "blob <size>\0<content>"
	expectedHeader := fmt.Sprintf("blob %d\x00", len(content))
	expectedBlob := append([]byte(expectedHeader), []byte(content)...)

	if !bytes.Equal(decompressed, expectedBlob) {
		t.Errorf("decompressed content mismatch:\n  got:  %q\n  want: %q", decompressed, expectedBlob)
	}
}

func TestWriteBlobWithSHA_NonExistentFile(t *testing.T) {
	root := t.TempDir()
	setupPurrObjectsDir(t, root)

	_, err := WriteBlobWithSHA(root, filepath.Join(root, "does-not-exist.txt"))
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}

func TestWriteBlobWithSHA_EmptyFile(t *testing.T) {
	root := t.TempDir()
	setupPurrObjectsDir(t, root)

	filePath := writeTestFile(t, root, "empty.txt", "")

	// Expected hash: SHA-1 of "blob 0\0"
	expectedBlob := []byte("blob 0\x00")
	expectedHash := sha1.Sum(expectedBlob)

	gotHash, err := WriteBlobWithSHA(root, filePath)
	if err != nil {
		t.Fatalf("WriteBlobWithSHA() unexpected error for empty file: %v", err)
	}

	if gotHash != expectedHash {
		t.Errorf("empty file hash mismatch:\n  got:  %x\n  want: %x", gotHash, expectedHash)
	}

	// Verify object was actually stored
	hashStr := fmt.Sprintf("%x", gotHash)
	objPath := filepath.Join(root, ".purr", "objects", hashStr[:2], hashStr[2:])
	if _, err := os.Stat(objPath); err != nil {
		t.Errorf("expected object file for empty blob, got error: %v", err)
	}
}

func TestComputeTreeSHA1_DeterministicHash(t *testing.T) {
	entries := []*TreeEntries{
		{Mode: "100644", Name: "file1.txt", Sha1Hex: strings.Repeat("a", 40)},
		{Mode: "100644", Name: "file2.txt", Sha1Hex: strings.Repeat("b", 40)},
	}

	hash1, err := ComputeTreeSHA1(".", entries)
	if err != nil {
		t.Fatalf("first ComputeTreeSHA1() error: %v", err)
	}

	// Rebuild entries to avoid any state from sorting side-effects
	entries2 := []*TreeEntries{
		{Mode: "100644", Name: "file1.txt", Sha1Hex: strings.Repeat("a", 40)},
		{Mode: "100644", Name: "file2.txt", Sha1Hex: strings.Repeat("b", 40)},
	}

	hash2, err := ComputeTreeSHA1(".", entries2)
	if err != nil {
		t.Fatalf("second ComputeTreeSHA1() error: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("deterministic check failed: %s != %s", hash1, hash2)
	}

	// Sanity: hash should be 40 hex characters
	if len(hash1) != 40 {
		t.Errorf("expected 40-character hex hash, got length %d: %s", len(hash1), hash1)
	}
}

func TestComputeTreeSHA1_DifferentEntries_DifferentHash(t *testing.T) {
	entriesA := []*TreeEntries{
		{Mode: "100644", Name: "alpha.txt", Sha1Hex: strings.Repeat("a", 40)},
	}
	entriesB := []*TreeEntries{
		{Mode: "100644", Name: "beta.txt", Sha1Hex: strings.Repeat("b", 40)},
	}

	hashA, err := ComputeTreeSHA1(".", entriesA)
	if err != nil {
		t.Fatalf("ComputeTreeSHA1(A) error: %v", err)
	}

	hashB, err := ComputeTreeSHA1(".", entriesB)
	if err != nil {
		t.Fatalf("ComputeTreeSHA1(B) error: %v", err)
	}

	if hashA == hashB {
		t.Errorf("expected different hashes for different entries, both got: %s", hashA)
	}
}

func TestComputeTreeSHA1_EmptyEntries_Error(t *testing.T) {
	_, err := ComputeTreeSHA1(".", []*TreeEntries{})
	if err == nil {
		t.Errorf("expected error for empty entries, got nil")
	}
}
