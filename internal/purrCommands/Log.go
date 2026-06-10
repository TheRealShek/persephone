package purrCommands

import (
	"Persephone/internal/objects"
	"Persephone/internal/refs"
	"Persephone/internal/ui"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogCommits renders the first-parent history reachable from HEAD.
//
// History Model:
// Persephone currently records one ParentHash per commit and does not create merge commits. Walking
// from HEAD toward the root therefore produces a linear newest-to-oldest history. The visited set is
// still required: loose objects are repository data, and a corrupted parent cycle must fail instead
// of keeping the CLI alive forever.
func LogCommits(rootDir string, out io.Writer) error {
	purrDir := filepath.Join(rootDir, ".purr")
	if _, err := os.Stat(purrDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not a purr repository")
		}
		return fmt.Errorf("failed to inspect repository: %w", err)
	}

	commitHash, err := refs.GetHEADCommit(rootDir)
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}
	if commitHash == "" {
		fmt.Fprintln(out, ui.Metadata("No commits yet"))
		return nil
	}

	visited := make(map[string]struct{})
	first := true

	for commitHash != "" {
		if _, exists := visited[commitHash]; exists {
			return fmt.Errorf("invalid commit history: cycle detected at %s", commitHash)
		}
		visited[commitHash] = struct{}{}

		commit, err := objects.ReadCommitObject(rootDir, commitHash)
		if err != nil {
			return fmt.Errorf("failed to read history at %s: %w", commitHash, err)
		}

		if !first {
			fmt.Fprintln(out)
		}
		printCommit(out, commitHash, commit)

		first = false
		commitHash = commit.ParentHash
	}

	return nil
}

func printCommit(out io.Writer, hash string, commit *objects.CommitObj) {
	fmt.Fprintln(out, ui.LogCommitHeader(hash))
	fmt.Fprintf(out, "%s %s <%s>\n", ui.LogLabel("Author:"), commit.Author.UserName, commit.Author.UserEmail)
	fmt.Fprintf(out, "%s   %s\n\n", ui.LogLabel("Date:"), commit.Timestamp.UTC().Format(time.RFC1123Z))
	fmt.Fprintln(out, ui.LogMessage(commit.Message))
}
