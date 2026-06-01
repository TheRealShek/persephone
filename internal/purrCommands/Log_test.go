package purrCommands_test

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/testutils"
	"Persephone/internal/utils"
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func storeLogCommit(t *testing.T, rootDir, hash, parentHash, message string) {
	t.Helper()

	commit := &utils.CommitObj{
		TreeHash:   strings.Repeat("a", 40),
		ParentHash: parentHash,
		Author:     utils.PurrConfig{UserName: "Log Tester", UserEmail: "log@example.com"},
		Committer:  utils.PurrConfig{UserName: "Log Tester", UserEmail: "log@example.com"},
		Message:    message,
		Timestamp:  time.Date(2026, 6, 1, 9, 30, 0, 0, time.UTC),
	}
	object, err := utils.BuildCommitObject(commit)
	if err != nil {
		t.Fatalf("BuildCommitObject() error = %v", err)
	}

	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(object); err != nil {
		t.Fatalf("failed to compress test commit: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to finalize test commit compression: %v", err)
	}

	objectDir := filepath.Join(rootDir, ".purr", "objects", hash[:2])
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		t.Fatalf("failed to create test object directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(objectDir, hash[2:]), compressed.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test commit: %v", err)
	}
}

func updateTestHEAD(t *testing.T, rootDir, hash string) {
	t.Helper()

	refPath := filepath.Join(rootDir, ".purr", "refs", "heads", "main")
	if err := os.WriteFile(refPath, []byte(hash+"\n"), 0644); err != nil {
		t.Fatalf("failed to update test HEAD: %v", err)
	}
}

func TestLogCommits_EmptyHistory(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	var out bytes.Buffer

	if err := purrCommands.LogCommits(repo, &out); err != nil {
		t.Fatalf("LogCommits() error = %v", err)
	}
	if got := out.String(); got != "No commits yet\n" {
		t.Fatalf("LogCommits() output = %q, want empty-history message", got)
	}
}

func TestLogCommits_NewestToOldest(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	firstHash := strings.Repeat("1", 40)
	secondHash := strings.Repeat("2", 40)
	storeLogCommit(t, repo, firstHash, "", "first commit")
	storeLogCommit(t, repo, secondHash, firstHash, "second commit\nwith details")
	updateTestHEAD(t, repo, secondHash)

	var out bytes.Buffer
	if err := purrCommands.LogCommits(repo, &out); err != nil {
		t.Fatalf("LogCommits() error = %v", err)
	}

	got := out.String()
	secondIndex := strings.Index(got, secondHash)
	firstIndex := strings.Index(got, firstHash)
	if secondIndex == -1 || firstIndex == -1 || secondIndex > firstIndex {
		t.Fatalf("LogCommits() did not render newest-to-oldest history:\n%s", got)
	}
	if !strings.Contains(got, "Author: Log Tester <log@example.com>") {
		t.Fatalf("LogCommits() output missing author metadata:\n%s", got)
	}
	if !strings.Contains(got, "    with details") {
		t.Fatalf("LogCommits() output missing indented multiline message:\n%s", got)
	}
}

func TestLogCommits_RejectsParentCycle(t *testing.T) {
	repo := testutils.SetupTestRepo(t)
	firstHash := strings.Repeat("a", 40)
	secondHash := strings.Repeat("b", 40)
	storeLogCommit(t, repo, firstHash, secondHash, "first commit")
	storeLogCommit(t, repo, secondHash, firstHash, "second commit")
	updateTestHEAD(t, repo, firstHash)

	var out bytes.Buffer
	err := purrCommands.LogCommits(repo, &out)
	if err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("LogCommits() error = %v, want cycle detection error", err)
	}
}

func TestLogCommits_RejectsNonRepository(t *testing.T) {
	var out bytes.Buffer
	err := purrCommands.LogCommits(t.TempDir(), &out)
	if err == nil || !strings.Contains(err.Error(), "not a purr repository") {
		t.Fatalf("LogCommits() error = %v, want repository error", err)
	}
}
