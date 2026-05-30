package colors

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	AnsiReset  = "\x1b[0m"
	AnsiBold   = "\x1b[1m"
	AnsiDim    = "\x1b[2m"
	AnsiGray   = "\x1b[90m"
	AnsiRed    = "\x1b[31m"
	AnsiGreen  = "\x1b[32m"
	AnsiYellow = "\x1b[33m"
)

type ColorSupport struct {
	Stdout bool
	Stderr bool
}

var Colors = DetectColorSupport()

func DetectColorSupport() ColorSupport {
	return ColorSupport{
		Stdout: SupportsColor(os.Stdout),
		Stderr: SupportsColor(os.Stderr),
	}
}

func SupportsColor(f *os.File) bool {
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

func Style(enabled bool, code string, s string) string {
	if !enabled || code == "" || s == "" {
		return s
	}
	return code + s + AnsiReset
}

func WroteLabel() string { return Style(Colors.Stdout, AnsiBold+AnsiGreen, "Wrote:") }
func NoteLabel() string  { return Style(Colors.Stderr, AnsiBold+AnsiYellow, "Note:") }
func ErrLabel() string   { return Style(Colors.Stderr, AnsiBold+AnsiRed, "Error:") }

func WarnText(s string) string { return Style(Colors.Stderr, AnsiYellow, s) }
func DimText(s string) string  { return Style(Colors.Stderr, AnsiDim+AnsiGray, s) }

func ExitStatusLine(code int) string {
	return DimText(fmt.Sprintf("exit status %d", code))
}
