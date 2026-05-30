package main

import (
	"os"
	"path/filepath"
)

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func repoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	dir := cwd
	for {
		// Heuristic: repo root contains these files.
		if fileExists(filepath.Join(dir, "package.json")) && fileExists(filepath.Join(dir, "README.md")) {
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
