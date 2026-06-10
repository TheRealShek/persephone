package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- Original tests (preserved) ----------

func TestReadWriteIndexRoundtrip(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	// Create test entries
	time1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	entries := []IndexEntry{
		{
			Ctime: time1,
			Mtime: time2,
			Dev:   1,
			Ino:   2,
			Mode:  3,
			Uid:   4,
			Gid:   5,
			Size:  6,
			Sha1:  [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			Stage: 0,
			Path:  "file1.txt",
		},
		{
			Ctime: time1,
			Mtime: time2,
			Dev:   10,
			Ino:   20,
			Mode:  30,
			Uid:   40,
			Gid:   50,
			Size:  60,
			Sha1:  [20]byte{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			Stage: 0,
			Path:  "dir/file2.txt",
		},
	}

	// Write
	err := WriteIndex(indexPath, entries)
	if err != nil {
		t.Fatalf("WriteIndex failed: %v", err)
	}

	// Read
	readEntries, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}

	// Compare
	if len(readEntries) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(readEntries))
	}

	for i := range entries {
		// Version 3 preserves nanosecond precision through {uint32 sec, uint32 nsec} pairs
		if !entries[i].Ctime.Equal(readEntries[i].Ctime) {
			t.Errorf("Entry %d Ctime mismatch: want %v, got %v", i, entries[i].Ctime, readEntries[i].Ctime)
		}
		if entries[i].Path != readEntries[i].Path {
			t.Errorf("Entry %d Path mismatch: expected %s, got %s", i, entries[i].Path, readEntries[i].Path)
		}
		if entries[i].Sha1 != readEntries[i].Sha1 {
			t.Errorf("Entry %d Sha1 mismatch", i)
		}
	}
}

func TestReadIndex_InvalidMagic(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	// Write invalid magic "BADD"
	badHeader := []byte("BADD\x00\x00\x00\x02\x00\x00\x00\x00")
	os.WriteFile(indexPath, badHeader, 0644)

	_, err := ReadIndex(indexPath)
	if err == nil {
		t.Fatal("Expected error when reading index with invalid magic bytes")
	}
	if !strings.Contains(err.Error(), "expected DIRC") {
		t.Errorf("Expected DIRC magic error, got: %v", err)
	}
}

func TestReadWriteIndex_LongPath(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	// Path close to 0xFFFF limit
	longPath := strings.Repeat("a", 4000)

	entries := []IndexEntry{
		{
			Ctime: time.Now(),
			Mtime: time.Now(),
			Path:  longPath,
		},
	}

	err := WriteIndex(indexPath, entries)
	if err != nil {
		t.Fatalf("WriteIndex failed on long path: %v", err)
	}

	readEntries, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex failed on long path: %v", err)
	}

	if readEntries[0].Path != longPath {
		t.Errorf("Long path mismatch")
	}
}

func TestWriteIndex_PaddingAlignment(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	// 64 fixed + 4 path bytes = 68.
	// Version 3 padding: 8 - (68 % 8) = 8 - 4 = 4 bytes padding (72 total entry).
	entries := []IndexEntry{
		{Path: "1234"},
	}

	WriteIndex(indexPath, entries)
	data, _ := os.ReadFile(indexPath)

	// 12 byte header + 72 byte entry = 84 bytes total.
	if len(data) != 84 {
		t.Errorf("Padding calculation incorrect. Expected 84 bytes, got %d", len(data))
	}
}

// ---------- New tests ----------

func TestWriteReadIndex_EmptyEntries(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	err := WriteIndex(indexPath, []IndexEntry{})
	if err != nil {
		t.Fatalf("WriteIndex with empty entries failed: %v", err)
	}

	// File should just be the 12-byte header
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}
	if len(data) != 12 {
		t.Errorf("Expected 12-byte header for empty index, got %d bytes", len(data))
	}

	// ReadIndex should return empty slice
	readEntries, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex on empty index failed: %v", err)
	}
	if len(readEntries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(readEntries))
	}
}

func TestReadIndex_NonExistentFile(t *testing.T) {
	_, err := ReadIndex("/nonexistent/path/index")
	if err == nil {
		t.Fatal("Expected error when reading non-existent index file")
	}
}

func TestReadIndex_TruncatedHeader(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	// Write only 8 bytes instead of 12
	os.WriteFile(indexPath, []byte("DIRC\x00\x00"), 0644)

	_, err := ReadIndex(indexPath)
	if err == nil {
		t.Fatal("Expected error when reading truncated index header")
	}
}

func TestReadWriteIndex_AllFieldsPreserved(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	ctime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	mtime := time.Date(2025, 6, 15, 11, 45, 0, 0, time.UTC)

	original := IndexEntry{
		Ctime: ctime,
		Mtime: mtime,
		Dev:   2049,
		Ino:   131072,
		Mode:  0100644,
		Uid:   1000,
		Gid:   1000,
		Size:  42,
		Sha1:  [20]byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		Stage: 0,
		Path:  "src/main.go",
	}

	err := WriteIndex(indexPath, []IndexEntry{original})
	if err != nil {
		t.Fatalf("WriteIndex failed: %v", err)
	}

	readEntries, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}

	if len(readEntries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(readEntries))
	}

	got := readEntries[0]

	// Verify every field
	if !original.Ctime.Equal(got.Ctime) {
		t.Errorf("Ctime mismatch: want %v, got %v", original.Ctime, got.Ctime)
	}
	if !original.Mtime.Equal(got.Mtime) {
		t.Errorf("Mtime mismatch: want %v, got %v", original.Mtime, got.Mtime)
	}
	if original.Dev != got.Dev {
		t.Errorf("Dev mismatch: want %d, got %d", original.Dev, got.Dev)
	}
	if original.Ino != got.Ino {
		t.Errorf("Ino mismatch: want %d, got %d", original.Ino, got.Ino)
	}
	if original.Mode != got.Mode {
		t.Errorf("Mode mismatch: want %o, got %o", original.Mode, got.Mode)
	}
	if original.Uid != got.Uid {
		t.Errorf("Uid mismatch: want %d, got %d", original.Uid, got.Uid)
	}
	if original.Gid != got.Gid {
		t.Errorf("Gid mismatch: want %d, got %d", original.Gid, got.Gid)
	}
	if original.Size != got.Size {
		t.Errorf("Size mismatch: want %d, got %d", original.Size, got.Size)
	}
	if original.Sha1 != got.Sha1 {
		t.Errorf("Sha1 mismatch: want %x, got %x", original.Sha1, got.Sha1)
	}
	if original.Stage != got.Stage {
		t.Errorf("Stage mismatch: want %d, got %d", original.Stage, got.Stage)
	}
	if original.Path != got.Path {
		t.Errorf("Path mismatch: want %q, got %q", original.Path, got.Path)
	}
}

func TestReadWriteIndex_MultipleEntries(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	now := time.Now()
	entries := []IndexEntry{
		{Ctime: now, Mtime: now, Path: "a.txt", Sha1: [20]byte{1}},
		{Ctime: now, Mtime: now, Path: "b/c.txt", Sha1: [20]byte{2}},
		{Ctime: now, Mtime: now, Path: "d/e/f.txt", Sha1: [20]byte{3}},
		{Ctime: now, Mtime: now, Path: "z.txt", Sha1: [20]byte{4}},
	}

	err := WriteIndex(indexPath, entries)
	if err != nil {
		t.Fatalf("WriteIndex failed: %v", err)
	}

	readEntries, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}

	if len(readEntries) != 4 {
		t.Fatalf("Expected 4 entries, got %d", len(readEntries))
	}

	// Verify paths are preserved in order
	expectedPaths := []string{"a.txt", "b/c.txt", "d/e/f.txt", "z.txt"}
	for i, expected := range expectedPaths {
		if readEntries[i].Path != expected {
			t.Errorf("Entry %d: expected path %q, got %q", i, expected, readEntries[i].Path)
		}
	}
}

func TestWriteIndex_PaddingVariousLengths(t *testing.T) {
	// Test that padding works correctly for various path lengths
	// Version 3: entry = 64 (fixed) + len(path) + padding (1-8 NUL bytes)
	// Total entry must be aligned to 8 bytes AND path must be NUL-terminated
	tests := []struct {
		name       string
		pathLen    int
		expectedSz int // total file size = 12 (header) + entry size
	}{
		{"path_len_1", 1, 12 + 72},   // 64+1=65, pad 7 -> 72
		{"path_len_2", 2, 12 + 72},   // 64+2=66, pad 6 -> 72
		{"path_len_6", 6, 12 + 72},   // 64+6=70, pad 2 -> 72
		{"path_len_7", 7, 12 + 72},   // 64+7=71, pad 1 -> 72
		{"path_len_8", 8, 12 + 80},   // 64+8=72, pad 8 -> 80 (NUL termination requires min 1 byte)
		{"path_len_9", 9, 12 + 80},   // 64+9=73, pad 7 -> 80
		{"path_len_16", 16, 12 + 88}, // 64+16=80, pad 8 -> 88 (NUL termination requires min 1 byte)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			indexPath := filepath.Join(tempDir, "index")

			path := strings.Repeat("x", tt.pathLen)
			entries := []IndexEntry{{Path: path}}

			WriteIndex(indexPath, entries)
			data, _ := os.ReadFile(indexPath)

			if len(data) != tt.expectedSz {
				t.Errorf("Path len %d: expected file size %d, got %d", tt.pathLen, tt.expectedSz, len(data))
			}
		})
	}
}

func TestReadWriteIndex_StageField(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index")

	now := time.Now()
	entries := []IndexEntry{
		{Ctime: now, Mtime: now, Path: "conflict.txt", Stage: 1},
		{Ctime: now, Mtime: now, Path: "normal.txt", Stage: 0},
	}

	err := WriteIndex(indexPath, entries)
	if err != nil {
		t.Fatalf("WriteIndex failed: %v", err)
	}

	readEntries, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex failed: %v", err)
	}

	if readEntries[0].Stage != 1 {
		t.Errorf("Expected stage 1, got %d", readEntries[0].Stage)
	}
	if readEntries[1].Stage != 0 {
		t.Errorf("Expected stage 0, got %d", readEntries[1].Stage)
	}
}
