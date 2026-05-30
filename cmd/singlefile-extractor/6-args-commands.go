package main

import (
	"io"
	"strings"
	"fmt"
)

type commandHelp struct {
	Name string
	Desc string
}

var allCommands = []commandHelp{
	{Name: "extract", Desc: "Extract a <form id=...> from nested iframe[srcdoc] HTML"},
	{Name: "moveout-css", Desc: "Move inline <style> blocks into a separate .css file"},
	{Name: "format-html", Desc: "Pretty-print HTML (optionally runs a CSS pipeline)"},
	{Name: "format-css", Desc: "Pretty-print CSS (optionally extracts url(data:...) to vars)"},
	{Name: "extract-data-urls", Desc: "Extract url(data:...) into a vars CSS file + rewrite CSS"},
}

func otherCommands(current string) []commandHelp {
	out := make([]commandHelp, 0, len(allCommands))
	for _, c := range allCommands {
		if c.Name == current {
			continue
		}
		out = append(out, c)
	}
	return out
}

func writeCommandTable(w io.Writer, commands []commandHelp) {
	maxName := 0
	for _, c := range commands {
		if len(c.Name) > maxName {
			maxName = len(c.Name)
		}
	}
	colWidth := maxName + 2

	for _, c := range commands {
		name := strings.TrimSpace(c.Name)
		desc := strings.TrimSpace(c.Desc)
		if name == "" {
			continue
		}
		if desc == "" {
			fmt.Fprintf(w, "  %s\n", name)
			continue
		}
		fmt.Fprintf(w, "  %-*s %s\n", colWidth, name, desc)
	}
}

