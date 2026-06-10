package purrcommands_test

import (
	"persephone/internal/purrcommands"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitPurrDirectories_CreatesAllDirs(t *testing.T) {
	base := t.TempDir()

	if err := purrcommands.InitPurrDirectories(base); err != nil {
		t.Fatalf("InitPurrDirectories() error = %v", err)
	}

	expectedDirs := []string{
		filepath.Join(".purr", "objects"),
		filepath.Join(".purr", "refs", "heads"),
		filepath.Join(".purr", "logs"),
	}
	for _, rel := range expectedDirs {
		dir := filepath.Join(base, rel)
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %s to exist, got error: %v", rel, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory, but it is not", rel)
		}
	}
}

func TestInitPurrDirectories_IndexFileHeader(t *testing.T) {
	base := t.TempDir()

	if err := purrcommands.InitPurrDirectories(base); err != nil {
		t.Fatalf("InitPurrDirectories() error = %v", err)
	}

	indexPath := filepath.Join(base, ".purr", "index")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index file: %v", err)
	}

	// Must be exactly 12 bytes
	if len(data) != 12 {
		t.Fatalf("expected index file to be 12 bytes, got %d", len(data))
	}

	// First 4 bytes: DIRC magic
	if string(data[:4]) != "DIRC" {
		t.Errorf("expected magic 'DIRC', got %q", string(data[:4]))
	}

	// Bytes 4-7: version 2 (big-endian uint32)
	version := binary.BigEndian.Uint32(data[4:8])
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}

	// Bytes 8-11: entry count 0 (big-endian uint32)
	entryCount := binary.BigEndian.Uint32(data[8:12])
	if entryCount != 0 {
		t.Errorf("expected 0 entries, got %d", entryCount)
	}
}

func TestInitPurrDirectories_HEADContent(t *testing.T) {
	base := t.TempDir()

	if err := purrcommands.InitPurrDirectories(base); err != nil {
		t.Fatalf("InitPurrDirectories() error = %v", err)
	}

	headPath := filepath.Join(base, ".purr", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("failed to read HEAD file: %v", err)
	}

	expected := "ref: refs/heads/main\n"
	if string(data) != expected {
		t.Errorf("expected HEAD content %q, got %q", expected, string(data))
	}
}

func TestInitPurrDirectories_RejectsExistingRepository(t *testing.T) {
	base := t.TempDir()

	// First call
	if err := purrcommands.InitPurrDirectories(base); err != nil {
		t.Fatalf("first InitPurrDirectories() error = %v", err)
	}

	// Capture file contents after first init
	indexPath := filepath.Join(base, ".purr", "index")
	headPath := filepath.Join(base, ".purr", "HEAD")

	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index after first init: %v", err)
	}
	headBefore, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("failed to read HEAD after first init: %v", err)
	}

	// Initialization is intentionally create-only. Existing metadata must be inspected and
	// repaired explicitly rather than modified as a side effect of a repeated command.
	if err := purrcommands.InitPurrDirectories(base); err == nil {
		t.Fatal("second InitPurrDirectories() expected an already-initialized error")
	}

	// Contents should be unchanged
	indexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index after second init: %v", err)
	}
	headAfter, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("failed to read HEAD after second init: %v", err)
	}

	if string(indexBefore) != string(indexAfter) {
		t.Error("index file was modified on second init")
	}
	if string(headBefore) != string(headAfter) {
		t.Error("HEAD file was modified on second init")
	}
}

func TestInitPurrDirectories_PreservesExistingIndex(t *testing.T) {
	base := t.TempDir()

	// Create .purr directory and a fake index with custom content
	purrDir := filepath.Join(base, ".purr")
	if err := os.MkdirAll(purrDir, 0755); err != nil {
		t.Fatalf("failed to create .purr dir: %v", err)
	}

	indexPath := filepath.Join(purrDir, "index")
	customContent := []byte("existing-index-data-with-entries")
	if err := os.WriteFile(indexPath, customContent, 0644); err != nil {
		t.Fatalf("failed to write custom index: %v", err)
	}

	// Init must reject an existing metadata root before touching its contents.
	if err := purrcommands.InitPurrDirectories(base); err == nil {
		t.Fatal("InitPurrDirectories() expected an already-initialized error")
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	if string(data) != string(customContent) {
		t.Errorf("existing index was overwritten: got %q, want %q", string(data), string(customContent))
	}
}

func TestInitPurrDirectories_PreservesExistingHEAD(t *testing.T) {
	base := t.TempDir()

	// Create .purr directory and a HEAD with custom content (e.g., pointing to a different branch)
	purrDir := filepath.Join(base, ".purr")
	if err := os.MkdirAll(purrDir, 0755); err != nil {
		t.Fatalf("failed to create .purr dir: %v", err)
	}

	headPath := filepath.Join(purrDir, "HEAD")
	customContent := []byte("ref: refs/heads/develop\n")
	if err := os.WriteFile(headPath, customContent, 0644); err != nil {
		t.Fatalf("failed to write custom HEAD: %v", err)
	}

	// Init must reject an existing metadata root before touching its contents.
	if err := purrcommands.InitPurrDirectories(base); err == nil {
		t.Fatal("InitPurrDirectories() expected an already-initialized error")
	}

	data, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("failed to read HEAD: %v", err)
	}

	if string(data) != string(customContent) {
		t.Errorf("existing HEAD was overwritten: got %q, want %q", string(data), string(customContent))
	}
}

func TestInitPurrDirectories_RejectsMetadataFile(t *testing.T) {
	base := t.TempDir()
	purrPath := filepath.Join(base, ".purr")
	if err := os.WriteFile(purrPath, []byte("occupied"), 0644); err != nil {
		t.Fatalf("failed to create metadata path fixture: %v", err)
	}

	err := purrcommands.InitPurrDirectories(base)
	if err == nil {
		t.Fatal("InitPurrDirectories() expected an error when .purr is a file")
	}
	if got := err.Error(); !strings.Contains(got, "is not a directory") {
		t.Fatalf("InitPurrDirectories() error = %q, want non-directory explanation", got)
	}
}

func TestReinitializePurrDirectories_RestoresMissingStructureAndPreservesMetadata(t *testing.T) {
	base := t.TempDir()
	purrDir := filepath.Join(base, ".purr")
	if err := os.MkdirAll(purrDir, 0755); err != nil {
		t.Fatalf("failed to create metadata root: %v", err)
	}

	indexContent := []byte("existing-index")
	headContent := []byte("ref: refs/heads/develop\n")
	if err := os.WriteFile(filepath.Join(purrDir, "index"), indexContent, 0644); err != nil {
		t.Fatalf("failed to write index fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(purrDir, "HEAD"), headContent, 0644); err != nil {
		t.Fatalf("failed to write HEAD fixture: %v", err)
	}

	if err := purrcommands.ReinitializePurrDirectories(base); err != nil {
		t.Fatalf("ReinitializePurrDirectories() error = %v", err)
	}

	for _, relPath := range []string{"objects", filepath.Join("refs", "heads"), "logs"} {
		info, err := os.Stat(filepath.Join(purrDir, relPath))
		if err != nil {
			t.Fatalf("expected restored directory %s: %v", relPath, err)
		}
		if !info.IsDir() {
			t.Fatalf("restored path %s is not a directory", relPath)
		}
	}

	indexAfter, err := os.ReadFile(filepath.Join(purrDir, "index"))
	if err != nil {
		t.Fatalf("failed to read index after reinitialize: %v", err)
	}
	headAfter, err := os.ReadFile(filepath.Join(purrDir, "HEAD"))
	if err != nil {
		t.Fatalf("failed to read HEAD after reinitialize: %v", err)
	}
	if string(indexAfter) != string(indexContent) {
		t.Fatalf("index was overwritten: got %q, want %q", indexAfter, indexContent)
	}
	if string(headAfter) != string(headContent) {
		t.Fatalf("HEAD was overwritten: got %q, want %q", headAfter, headContent)
	}
}

func TestReinitializePurrDirectories_RejectsMissingRepository(t *testing.T) {
	if err := purrcommands.ReinitializePurrDirectories(t.TempDir()); err == nil {
		t.Fatal("ReinitializePurrDirectories() expected an error for missing metadata root")
	}
}
