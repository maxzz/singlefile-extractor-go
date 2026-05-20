package main

import (
	"fmt"
	"io"
)

func isHelpArg(s string) bool {
	return s == "-h" || s == "--help" || s == "-help"
}

func printRootUsage(w io.Writer) {
	fmt.Fprintf(w, "singlefile-extractor utilities (Go) v%s\n", version)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  singlefile-extractor <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	writeCommandTable(w, allCommands)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `Run "<command> --help" for command-specific flags.`)
}
