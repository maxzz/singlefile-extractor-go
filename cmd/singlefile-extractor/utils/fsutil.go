package utils

import (
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func RepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	dir := cwd
	for {
		// Heuristic: repo root contains these files.
		if FileExists(filepath.Join(dir, "package.json")) && FileExists(filepath.Join(dir, "README.md")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}
