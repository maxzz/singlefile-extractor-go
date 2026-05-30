package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiGray   = "\x1b[90m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
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
	// Standard opt-out.
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Standard opt-in used by many CLIs (chalk/picocolors-style).
	if v, ok := os.LookupEnv("FORCE_COLOR"); ok {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "" || v == "1" || v == "true" || v == "yes" || v == "on" {
			// Still require actual terminal support; on Windows this will try to
			// enable virtual terminal sequences.
			if runtime.GOOS == "windows" {
				return enableVirtualTerminalProcessing(f)
			}
			return true
		}
		if v == "0" || v == "false" || v == "no" || v == "off" {
			return false
		}
		// Any other value: treat as enabled (but still require terminal support).
		if runtime.GOOS == "windows" {
			return enableVirtualTerminalProcessing(f)
		}
		return true
	}

	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" {
		return false
	}

	// If we're not attached to a terminal, keep output clean by default.
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		// If stat fails, be conservative.
		return false
	}
	if (info.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	// On Windows, cmd.exe requires enabling virtual terminal processing for ANSI
	// escapes to render as colors. If this fails, disable colors to avoid printing
	// raw escape codes.
	if runtime.GOOS == "windows" {
		return enableVirtualTerminalProcessing(f)
	}

	return true
}

func style(enabled bool, code string, s string) string {
	if !enabled || code == "" || s == "" {
		return s
	}
	return code + s + ansiReset
}

func wroteLabel() string { return style(colors.stdout, ansiBold+ansiGreen, "Wrote:") }
func noteLabel() string  { return style(colors.stderr, ansiBold+ansiYellow, "Note:") }
func errLabel() string   { return style(colors.stderr, ansiBold+ansiRed, "Error:") }

func warnText(s string) string { return style(colors.stderr, ansiYellow, s) }
func dimText(s string) string  { return style(colors.stderr, ansiDim+ansiGray, s) }

func exitStatusLine(code int) string {
	return dimText(fmt.Sprintf("exit status %d", code))
}
