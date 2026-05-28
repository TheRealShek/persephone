package utils

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildTreeObject(t *testing.T) {
	tests := []struct {
		name        string
		entries     []*TreeEntries
		expectError bool
		checkOut    func(t *testing.T, out []byte)
	}{
		{
			name:        "Empty entries",
			entries:     []*TreeEntries{},
			expectError: true,
		},
		{
			name: "Invalid mode",
			entries: []*TreeEntries{
				{Mode: "123456", Name: "file.txt", Sha1Hex: "0123456789abcdef0123456789abcdef01234567"},
			},
			expectError: true,
		},
		{
			name: "Invalid SHA",
			entries: []*TreeEntries{
				{Mode: "100644", Name: "file.txt", Sha1Hex: "invalidhex"},
			},
			expectError: true,
		},
		{
			name: "Valid entries with sorting",
			entries: []*TreeEntries{
				{Mode: "100644", Name: "z-file.txt", Sha1Hex: strings.Repeat("a", 40)},
				{Mode: "040000", Name: "a-dir", IsTree: true, Sha1Hex: strings.Repeat("b", 40)},
			},
			expectError: false,
			checkOut: func(t *testing.T, out []byte) {
				// Header should be tree <size>\0
				if !bytes.HasPrefix(out, []byte("tree ")) {
					t.Errorf("Expected tree header, got: %s", out[:5])
				}
				// Verify sorting (a-dir/ should come before z-file.txt)
				idxA := bytes.Index(out, []byte("040000 a-dir\x00"))
				idxZ := bytes.Index(out, []byte("100644 z-file.txt\x00"))
				if idxA == -1 || idxZ == -1 || idxA > idxZ {
					t.Errorf("Entries not sorted correctly: a-dir idx=%d, z-file idx=%d", idxA, idxZ)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := BuildTreeObject(tt.entries)
			if (err != nil) != tt.expectError {
				t.Errorf("BuildTreeObject() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && tt.checkOut != nil {
				tt.checkOut(t, out)
			}
		})
	}
}

func TestBuildCommitObject(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		commit      *CommitObj
		expectError bool
		checkOut    func(t *testing.T, out []byte)
	}{
		{
			name: "Missing Tree Hash",
			commit: &CommitObj{
				Message:   "msg",
				Author:    PurrConfig{UserName: "A", UserEmail: "B"},
				Timestamp: testTime,
			},
			expectError: true,
		},
		{
			name: "Missing Timestamp",
			commit: &CommitObj{
				TreeHash: "treehash",
				Message:  "msg",
				Author:   PurrConfig{UserName: "A", UserEmail: "B"},
			},
			expectError: true,
		},
		{
			name: "Valid commit",
			commit: &CommitObj{
				TreeHash:   "tree123",
				ParentHash: "parent123",
				Author:     PurrConfig{UserName: "Jane Doe", UserEmail: "jane@example.com"},
				Committer:  PurrConfig{UserName: "Jane Doe", UserEmail: "jane@example.com"},
				Message:    "Initial commit",
				Timestamp:  testTime,
			},
			expectError: false,
			checkOut: func(t *testing.T, out []byte) {
				content := string(out)
				if !strings.HasPrefix(content, "commit ") {
					t.Errorf("Missing commit header")
				}
				if !strings.Contains(content, "tree tree123\n") {
					t.Errorf("Missing tree hash line")
				}
				if !strings.Contains(content, "parent parent123\n") {
					t.Errorf("Missing parent hash line")
				}
				timeStr := fmt.Sprintf("%d +0000", testTime.Unix())
				if !strings.Contains(content, timeStr) {
					t.Errorf("Missing correct timestamp")
				}
				if !strings.HasSuffix(content, "\n\nInitial commit\n") {
					t.Errorf("Message formatting incorrect")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := BuildCommitObject(tt.commit)
			if (err != nil) != tt.expectError {
				t.Errorf("BuildCommitObject() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && tt.checkOut != nil {
				tt.checkOut(t, out)
			}
		})
	}
}
