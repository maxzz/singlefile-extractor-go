package main

import (
	"fmt"
	"io"
)

func isHelpArg(s string) bool {
	return s == "-h" || s == "--help" || s == "-help"
}

func printRootUsage(w io.Writer) {
	fmt.Fprint(w, `singlefile-extractor utilities (Go)

Usage:
  singlefile-extractor <command> [options]

Commands:
  extract             Extract a <form id=...> from nested iframe[srcdoc] HTML
  moveout-css         Move inline <style> blocks into a separate .css file
  format-html         Pretty-print HTML (optionally runs a CSS pipeline)
  format-css          Pretty-print CSS (optionally extracts url(data:...) to vars)
  extract-data-urls   Extract url(data:...) into a vars CSS file + rewrite CSS

Run "<command> --help" for command-specific flags.
`)
}

