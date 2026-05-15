package main

import (
	"os"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

type colorSupport struct {
	stdout bool
	stderr bool
}

var colors = detectColorSupport()

func detectColorSupport() colorSupport {
	return colorSupport{
		stdout: supportsColor(os.Stdout),
		stderr: supportsColor(os.Stderr),
	}
}

func supportsColor(f *os.File) bool {
	// Standard opt-out used by many CLIs.
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" {
		return false
	}

	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	// Only colorize when writing to an interactive terminal.
	return (info.Mode() & os.ModeCharDevice) != 0
}

func style(enabled bool, codes string, s string) string {
	if !enabled || codes == "" || s == "" {
		return s
	}
	return codes + s + ansiReset
}

func outLabelSuccess(s string) string { return style(colors.stdout, ansiBold+ansiGreen, s) }
func outLabelInfo(s string) string    { return style(colors.stdout, ansiBold+ansiCyan, s) }
func outDim(s string) string          { return style(colors.stdout, ansiDim, s) }

func errLabelWarn(s string) string  { return style(colors.stderr, ansiBold+ansiYellow, s) }
func errLabelError(s string) string { return style(colors.stderr, ansiBold+ansiRed, s) }

