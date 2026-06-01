package utils

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func storeCompressedObject(t *testing.T, rootDir, hash string, object []byte) {
	t.Helper()

	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(object); err != nil {
		t.Fatalf("failed to compress test object: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to finalize test object compression: %v", err)
	}

	objectDir := filepath.Join(rootDir, ".purr", "objects", hash[:2])
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		t.Fatalf("failed to create test object directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(objectDir, hash[2:]), compressed.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test object: %v", err)
	}
}

func TestReadCommitObject_RoundTrip(t *testing.T) {
	rootDir := t.TempDir()
	timestamp := time.Date(2026, 6, 1, 9, 30, 0, 0, time.UTC)
	commit := &CommitObj{
		TreeHash:   strings.Repeat("a", 40),
		ParentHash: strings.Repeat("b", 40),
		Author:     PurrConfig{UserName: "Test Developer", UserEmail: "test@example.com"},
		Committer:  PurrConfig{UserName: "Test Developer", UserEmail: "test@example.com"},
		Message:    "subject line\nbody line",
		Timestamp:  timestamp,
	}

	object, err := BuildCommitObject(commit)
	if err != nil {
		t.Fatalf("BuildCommitObject() error = %v", err)
	}
	hash, err := ComputeCommitSHA1(commit)
	if err != nil {
		t.Fatalf("ComputeCommitSHA1() error = %v", err)
	}
	storeCompressedObject(t, rootDir, hash, object)

	got, err := ReadCommitObject(rootDir, hash)
	if err != nil {
		t.Fatalf("ReadCommitObject() error = %v", err)
	}

	if got.TreeHash != commit.TreeHash || got.ParentHash != commit.ParentHash {
		t.Fatalf("ReadCommitObject() links = (%q, %q), want (%q, %q)", got.TreeHash, got.ParentHash, commit.TreeHash, commit.ParentHash)
	}
	if got.Author != commit.Author || got.Committer != commit.Committer {
		t.Fatalf("ReadCommitObject() identities = (%+v, %+v), want (%+v, %+v)", got.Author, got.Committer, commit.Author, commit.Committer)
	}
	if got.Message != commit.Message {
		t.Fatalf("ReadCommitObject() message = %q, want %q", got.Message, commit.Message)
	}
	if !got.Timestamp.Equal(timestamp) {
		t.Fatalf("ReadCommitObject() timestamp = %v, want %v", got.Timestamp, timestamp)
	}
}

func TestReadCommitObject_RejectsInvalidHash(t *testing.T) {
	if _, err := ReadCommitObject(t.TempDir(), "short"); err == nil {
		t.Fatal("ReadCommitObject() expected an error for a malformed hash")
	}
}

func TestReadCommitObject_RejectsPayloadSizeMismatch(t *testing.T) {
	rootDir := t.TempDir()
	hash := strings.Repeat("c", 40)
	storeCompressedObject(t, rootDir, hash, []byte("commit 999\x00tree payload"))

	if _, err := ReadCommitObject(rootDir, hash); err == nil {
		t.Fatal("ReadCommitObject() expected an error for a payload size mismatch")
	}
}
