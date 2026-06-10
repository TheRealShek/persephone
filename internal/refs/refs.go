package refs

import (
	"os"
	"path/filepath"
	"strings"
)

func GetHEADCommit(rootDir string) (string, error) {
	headPath := filepath.Join(rootDir, ".purr", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	ref := strings.TrimSpace(string(content))
	if strings.HasPrefix(ref, "ref:") {
		branchRef := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		branchPath := filepath.Join(rootDir, ".purr", branchRef)
		hash, err := os.ReadFile(branchPath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		return strings.TrimSpace(string(hash)), nil
	}
	return ref, nil
}

func UpdateHEAD(rootDir string, commitHash string) error {
	headPath := filepath.Join(rootDir, ".purr", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return err
	}
	ref := strings.TrimSpace(string(content))
	if strings.HasPrefix(ref, "ref:") {
		branchRef := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		branchPath := filepath.Join(rootDir, ".purr", branchRef)
		tmpPath := branchPath + ".tmp"
		if err := os.WriteFile(tmpPath, []byte(commitHash+"\n"), 0644); err != nil {
			os.Remove(tmpPath)
			return err
		}
		if err := os.Rename(tmpPath, branchPath); err != nil {
			os.Remove(tmpPath)
			return err
		}
		return nil
	}
	tmpPath := headPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(commitHash+"\n"), 0644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, headPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
