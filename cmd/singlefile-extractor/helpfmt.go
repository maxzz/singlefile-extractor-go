package main

import (
	"fmt"
	"io"
	"strings"
)

type optionHelp struct {
	Short string // e.g. "i"
	Long  string // e.g. "input"
	Arg   string // e.g. "<path>" or "" for bool flags
	Desc  string // single-line description (include defaults/required here)
}

type usageSpec struct {
	Summary   string
	UsageLine string
	Options   []optionHelp
	Footer    string // optional (printed after options)
}

func printUsage(w io.Writer, spec usageSpec) {
	if strings.TrimSpace(spec.Summary) != "" {
		fmt.Fprintln(w, spec.Summary)
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n\n", strings.TrimSpace(spec.UsageLine))

	if len(spec.Options) > 0 {
		fmt.Fprintln(w, "Options:")
		maxLeft := 0
		lefts := make([]string, 0, len(spec.Options))
		for _, opt := range spec.Options {
			left := formatOptionLeft(opt)
			lefts = append(lefts, left)
			if len(left) > maxLeft {
				maxLeft = len(left)
			}
		}

		colWidth := maxLeft + 2
		for i, opt := range spec.Options {
			left := lefts[i]
			desc := strings.TrimSpace(opt.Desc)
			if desc == "" {
				fmt.Fprintf(w, "  %s\n", left)
				continue
			}
			fmt.Fprintf(w, "  %-*s %s\n", colWidth, left, desc)
		}
	}

	footer := strings.TrimSpace(spec.Footer)
	if footer != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, footer)
	}
}

func formatOptionLeft(opt optionHelp) string {
	parts := make([]string, 0, 2)
	if opt.Short != "" {
		parts = append(parts, "-"+opt.Short)
	}
	if opt.Long != "" {
		parts = append(parts, "--"+opt.Long)
	}

	left := strings.Join(parts, ", ")
	if strings.TrimSpace(opt.Arg) != "" {
		left += " " + strings.TrimSpace(opt.Arg)
	}
	return left
}

