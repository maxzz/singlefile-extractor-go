package main

import (
	"fmt"

	"singlefile-extractor-go/cmd/singlefile-extractor/colors"
)

func Style(enabled bool, code string, s string) string {
	if !enabled || code == "" || s == "" {
		return s
	}
	return code + s + colors.AnsiReset
}

func WroteLabel() string { return Style(colors.Colors.Stdout, colors.AnsiBold+colors.AnsiGreen, "Wrote:") }
func NoteLabel() string  { return Style(colors.Colors.Stderr, colors.AnsiBold+colors.AnsiYellow, "Note:") }
func ErrLabel() string   { return Style(colors.Colors.Stderr, colors.AnsiBold+colors.AnsiRed, "Error:") }

func WarnText(s string) string { return Style(colors.Colors.Stderr, colors.AnsiYellow, s) }
func DimText(s string) string  { return Style(colors.Colors.Stderr, colors.AnsiDim+colors.AnsiGray, s) }

func ExitStatusLine(code int) string {
	return DimText(fmt.Sprintf("exit status %d", code))
}
