package main

import (
	"fmt"
	"os"
	"strings"

	"singlefile-extractor-go/cmd/singlefile-extractor/colors"
)

// version is the current version of the utility, injected at build time using ldflags
var version = "0.1.0"

func main() {
	code := run(os.Args[1:])
	if code != 0 && shouldPrintExitStatusLine() {
		fmt.Fprintln(os.Stderr, colors.ExitStatusLine(code))
	}
	os.Exit(code)
}

func run(args []string) int {
	if len(args) == 0 || isHelpArg(args[0]) || args[0] == "help" {
		printRootUsage(os.Stdout)
		return 0
	}

	cmd := args[0]
	if cmd == "--version" || cmd == "-v" || cmd == "version" {
		fmt.Printf("singlefile-extractor utilities (Go) v%s\n", version)
		return 0
	}

	if strings.HasPrefix(cmd, "-") && hasValidInputFlag(args) {
		// Default command: if no command is provided but --input/-i is, treat it as
		// "format-html".
		return cmdFormatHTML(args)
	}

	subArgs := args[1:]

	switch cmd {
	case "extract":
		return cmdExtract(subArgs)
	case "moveout-css":
		return cmdMoveoutCSS(subArgs)
	case "format-html":
		return cmdFormatHTML(subArgs)
	case "format-css":
		return cmdFormatCSS(subArgs)
	case "extract-data-urls":
		return cmdExtractDataURLs(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "%s %s\n\n", colors.ErrLabel(), cmd)
		printRootUsage(os.Stderr)
		return 2
	}
}
