package main

import (
	"os"
	"path/filepath"
	"strings"
)

func shouldPrintExitStatusLine() bool {
	// When running via `go run`, the `go` tool prints its own "exit status N"
	// line after the program exits. Avoid duplicating it.
	return !isLikelyGoRunBinary()
}

func isLikelyGoRunBinary() bool {
	// Heuristic: go run executes a temp binary under a directory named "go-build*".
	// Example (Windows):
	//   %TEMP%\go-build123456789\b001\exe\main.exe
	args0 := os.Args[0]
	if strings.TrimSpace(args0) == "" {
		return false
	}

	a0 := filepath.ToSlash(strings.ToLower(args0))
	tmp := filepath.ToSlash(strings.ToLower(os.TempDir()))

	if tmp != "" && strings.Contains(a0, tmp) && strings.Contains(a0, "/go-build") {
		return true
	}
	// Fallback (covers non-standard temp dirs / env overrides).
	return strings.Contains(a0, "/go-build")
}

