package utils

import (
	"os"
	"path/filepath"
)

func ReadFileText(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func WriteFileText(path string, content string) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func WriteFileBytes(path string, content []byte) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
