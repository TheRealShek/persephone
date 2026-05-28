package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
		// Time objects read from binary are strictly Unix timestamps (seconds)
		// We should compare the Unix() value
		if entries[i].Ctime.Unix() != readEntries[i].Ctime.Unix() {
			t.Errorf("Entry %d Ctime mismatch", i)
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

	// 62 metadata + 2 pathlen + 4 path bytes = 68.
	// 68 % 8 = 4. Needs 4 bytes padding (72 total).
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
