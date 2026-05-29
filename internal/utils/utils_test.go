package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ExistsAndIsDirectory
// ---------------------------------------------------------------------------

func TestExistsAndIsDirectory_ExistingDir(t *testing.T) {
	dir := t.TempDir()

	isDir, err := ExistsAndIsDirectory(dir)
	if err != nil {
		t.Fatalf("ExistsAndIsDirectory(%q) unexpected error: %v", dir, err)
	}
	if !isDir {
		t.Errorf("ExistsAndIsDirectory(%q) = false, want true", dir)
	}
}

func TestExistsAndIsDirectory_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	isDir, err := ExistsAndIsDirectory(filePath)
	if err != nil {
		t.Fatalf("ExistsAndIsDirectory(%q) unexpected error: %v", filePath, err)
	}
	if isDir {
		t.Errorf("ExistsAndIsDirectory(%q) = true, want false (it's a file)", filePath)
	}
}

func TestExistsAndIsDirectory_NonExistent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	isDir, err := ExistsAndIsDirectory(missing)
	if err == nil {
		t.Fatal("ExistsAndIsDirectory() expected error for non-existent path, got nil")
	}
	if isDir {
		t.Errorf("ExistsAndIsDirectory() = true, want false for non-existent path")
	}
}

// ---------------------------------------------------------------------------
// WalkAndAddFiles
// ---------------------------------------------------------------------------

func TestWalkAndAddFiles_SkipsHidden(t *testing.T) {
	root := t.TempDir()

	// Hidden directory with a file inside
	hiddenDir := filepath.Join(root, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("failed to create hidden dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to write file in hidden dir: %v", err)
	}

	// Hidden file at root level
	if err := os.WriteFile(filepath.Join(root, ".dotfile"), []byte("dot"), 0644); err != nil {
		t.Fatalf("failed to write dotfile: %v", err)
	}

	// One visible file that should be collected
	if err := os.WriteFile(filepath.Join(root, "visible.txt"), []byte("visible"), 0644); err != nil {
		t.Fatalf("failed to write visible file: %v", err)
	}

	var collected []string
	err := WalkAndAddFiles(root, func(path string) error {
		collected = append(collected, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkAndAddFiles() unexpected error: %v", err)
	}

	if len(collected) != 1 {
		t.Fatalf("WalkAndAddFiles() collected %d files, want 1; got %v", len(collected), collected)
	}
	if !strings.HasSuffix(collected[0], "visible.txt") {
		t.Errorf("expected visible.txt, got %q", collected[0])
	}
}

func TestWalkAndAddFiles_CollectsRegularFiles(t *testing.T) {
	root := t.TempDir()

	files := []string{"a.txt", "b.txt", "subdir/c.txt"}
	for _, f := range files {
		full := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("failed to create directory for %s: %v", f, err)
		}
		if err := os.WriteFile(full, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}

	var collected []string
	err := WalkAndAddFiles(root, func(path string) error {
		rel, _ := filepath.Rel(root, path)
		collected = append(collected, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkAndAddFiles() unexpected error: %v", err)
	}

	sort.Strings(collected)
	want := []string{"a.txt", "b.txt", filepath.Join("subdir", "c.txt")}
	sort.Strings(want)

	if len(collected) != len(want) {
		t.Fatalf("collected %d files, want %d; got %v", len(collected), len(want), collected)
	}
	for i := range want {
		if collected[i] != want[i] {
			t.Errorf("collected[%d] = %q, want %q", i, collected[i], want[i])
		}
	}
}

func TestWalkAndAddFiles_SkipsDirectories(t *testing.T) {
	root := t.TempDir()

	// Create a nested structure
	if err := os.MkdirAll(filepath.Join(root, "dir1", "dir2"), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir1", "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var collected []string
	err := WalkAndAddFiles(root, func(path string) error {
		collected = append(collected, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkAndAddFiles() unexpected error: %v", err)
	}

	// Only the one file should appear; directories must not be passed to callback
	for _, p := range collected {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("os.Stat(%q) unexpected error: %v", p, err)
		}
		if info.IsDir() {
			t.Errorf("directory %q was passed to handleFile callback; want only files", p)
		}
	}
	if len(collected) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(collected), collected)
	}
}

// ---------------------------------------------------------------------------
// StoreObject
// ---------------------------------------------------------------------------

func TestStoreObject_CreatesDirectoryAndFile(t *testing.T) {
	root := t.TempDir()
	hash := "aabbccddee1122334455aabbccddee1122334455" // 40 hex chars
	data := []byte("blob content")

	if err := StoreObject(root, hash, data); err != nil {
		t.Fatalf("StoreObject() unexpected error: %v", err)
	}

	objectPath := filepath.Join(root, ".purr", "objects", hash[:2], hash[2:])
	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		t.Fatalf("StoreObject() did not create file at %s", objectPath)
	}
}

func TestStoreObject_ValidContent(t *testing.T) {
	root := t.TempDir()
	hash := "ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00"
	want := []byte("compressed blob data goes here")

	if err := StoreObject(root, hash, want); err != nil {
		t.Fatalf("StoreObject() unexpected error: %v", err)
	}

	objectPath := filepath.Join(root, ".purr", "objects", hash[:2], hash[2:])
	got, err := os.ReadFile(objectPath)
	if err != nil {
		t.Fatalf("failed to read stored object: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("stored content = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// PopulateAllIndexField
// ---------------------------------------------------------------------------

func TestPopulateAllIndexField(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "testfile.txt")
	content := []byte("hello world, this is test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("os.Stat() unexpected error: %v", err)
	}

	relPath := "testfile.txt"
	entry := PopulateAllIndexField(info, relPath)

	// Verify Path
	if entry.Path != relPath {
		t.Errorf("Path = %q, want %q", entry.Path, relPath)
	}

	// Verify Size matches content length
	if entry.Size != uint32(len(content)) {
		t.Errorf("Size = %d, want %d", entry.Size, len(content))
	}

	// Verify Mode is non-zero (valid file mode)
	if entry.Mode == 0 {
		t.Error("Mode = 0, want non-zero file mode")
	}

	// Verify Mtime is populated and matches info.ModTime()
	if !entry.Mtime.Equal(info.ModTime()) {
		t.Errorf("Mtime = %v, want %v", entry.Mtime, info.ModTime())
	}

	// Verify Ctime is not zero (platform.ExtractStat should populate it)
	if entry.Ctime.IsZero() {
		t.Error("Ctime is zero, want populated change time")
	}

	// Verify Stage is 0 (normal, non-conflict)
	if entry.Stage != 0 {
		t.Errorf("Stage = %d, want 0", entry.Stage)
	}

	// SHA1 should be zero-valued (PopulateAllIndexField doesn't compute it)
	var zeroSHA [20]byte
	if entry.Sha1 != zeroSHA {
		t.Errorf("Sha1 = %x, want zero value", entry.Sha1)
	}
}

// ---------------------------------------------------------------------------
// GetHEADCommit / UpdateHEAD helpers
// ---------------------------------------------------------------------------



func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dirs for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// ---------------------------------------------------------------------------
// GetHEADCommit
// ---------------------------------------------------------------------------

func TestGetHEADCommit_SymbolicRef(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	wantHash := "abc123def456abc123def456abc123def456abc1"
	writeFile(t, filepath.Join(root, ".purr", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(root, ".purr", "refs", "heads", "main"), wantHash+"\n")

	got, err := GetHEADCommit(root)
	if err != nil {
		t.Fatalf("GetHEADCommit() unexpected error: %v", err)
	}
	if got != wantHash {
		t.Errorf("GetHEADCommit() = %q, want %q", got, wantHash)
	}
}

func TestGetHEADCommit_DetachedHead(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	wantHash := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	writeFile(t, filepath.Join(root, ".purr", "HEAD"), wantHash+"\n")

	got, err := GetHEADCommit(root)
	if err != nil {
		t.Fatalf("GetHEADCommit() unexpected error: %v", err)
	}
	if got != wantHash {
		t.Errorf("GetHEADCommit() = %q, want %q", got, wantHash)
	}
}

func TestGetHEADCommit_NoHEAD(t *testing.T) {
	root := t.TempDir()
	// Don't create .purr/HEAD at all

	_, err := GetHEADCommit(root)
	if err == nil {
		t.Fatal("GetHEADCommit() expected error when HEAD is missing, got nil")
	}
}

// ---------------------------------------------------------------------------
// UpdateHEAD
// ---------------------------------------------------------------------------

func TestUpdateHEAD_SymbolicRef(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	// HEAD points to a branch
	writeFile(t, filepath.Join(root, ".purr", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(root, ".purr", "refs", "heads", "main"), "oldhash\n")

	newHash := "newhash123newhash123newhash123newhash1234"
	if err := UpdateHEAD(root, newHash); err != nil {
		t.Fatalf("UpdateHEAD() unexpected error: %v", err)
	}

	// Verify the branch file was updated, not HEAD itself
	branchContent, err := os.ReadFile(filepath.Join(root, ".purr", "refs", "heads", "main"))
	if err != nil {
		t.Fatalf("failed to read branch file: %v", err)
	}
	got := strings.TrimSpace(string(branchContent))
	if got != newHash {
		t.Errorf("branch ref = %q, want %q", got, newHash)
	}

	// HEAD should still be a symbolic ref
	headContent, err := os.ReadFile(filepath.Join(root, ".purr", "HEAD"))
	if err != nil {
		t.Fatalf("failed to read HEAD: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(headContent)), "ref:") {
		t.Errorf("HEAD should still be a symbolic ref, got %q", headContent)
	}
}

func TestUpdateHEAD_DetachedHead(t *testing.T) {
	root := t.TempDir()
	setupPurrDir(t, root)

	oldHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	writeFile(t, filepath.Join(root, ".purr", "HEAD"), oldHash+"\n")

	newHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := UpdateHEAD(root, newHash); err != nil {
		t.Fatalf("UpdateHEAD() unexpected error: %v", err)
	}

	headContent, err := os.ReadFile(filepath.Join(root, ".purr", "HEAD"))
	if err != nil {
		t.Fatalf("failed to read HEAD after update: %v", err)
	}
	got := strings.TrimSpace(string(headContent))
	if got != newHash {
		t.Errorf("HEAD = %q, want %q", got, newHash)
	}
}
