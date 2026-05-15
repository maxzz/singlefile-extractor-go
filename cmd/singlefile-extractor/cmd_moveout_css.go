package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var moveoutStyleBlockRe = regexp.MustCompile(`(?is)<style\b[^>]*>(.*?)</style>`)

func cmdMoveoutCSS(argv []string) int {
	root := repoRoot()
	defaultInput := filepath.Join(root, "tests", "esignature-form.html")

	var (
		inputPath   string
		outputPath  string
		cssOutPath  string
		href        string
		showHelp    bool
	)

	fs := flag.NewFlagSet("moveout-css", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprint(os.Stdout, `Move inline <style> CSS blocks into a separate .css file and link it from the HTML.

Usage:
  singlefile-extractor moveout-css [options]

Options:
`)
		fs.PrintDefaults()
	}

	fs.StringVar(&inputPath, "input", defaultInput, "Path to the HTML file to process.")
	fs.StringVar(&inputPath, "i", defaultInput, "Path to the HTML file to process.")
	fs.StringVar(&outputPath, "output", "", "Where to write the updated HTML (default: overwrite --input).")
	fs.StringVar(&outputPath, "o", "", "Where to write the updated HTML (default: overwrite --input).")
	fs.StringVar(&cssOutPath, "css-output", "", "Where to write extracted CSS (default: <output>.css).")
	fs.StringVar(&href, "href", "", "Optional href to use in the inserted <link> tag (default: relative path to --css-output).")
	fs.BoolVar(&showHelp, "help", false, "Show help.")
	fs.BoolVar(&showHelp, "h", false, "Show help.")

	if err := fs.Parse(argv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fs.Usage()
		return 2
	}
	if showHelp {
		fs.Usage()
		return 0
	}

	if outputPath == "" {
		outputPath = inputPath
	}
	if cssOutPath == "" {
		cssOutPath = replaceExt(outputPath, ".css")
	}

	htmlText, err := readFileText(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n%v\n", inputPath, err)
		return 1
	}

	cssChunks := extractStyleContentsMoveout(htmlText)
	if len(cssChunks) == 0 {
		fmt.Fprintf(os.Stderr, "No <style> blocks found in: %s\n", inputPath)
		return 1
	}

	cssText := strings.Join(cssChunks, "\n\n")
	cssText = strings.TrimRight(cssText, "\r\n") + "\n"

	if err := writeFileText(cssOutPath, cssText); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write CSS: %s\n%v\n", cssOutPath, err)
		return 1
	}

	htmlNoStyles := moveoutStyleBlockRe.ReplaceAllString(htmlText, "")
	hrefUse := href
	if hrefUse == "" {
		hrefUse = computeDefaultHref(outputPath, cssOutPath)
	}
	htmlOut := insertStylesheetLinkSimple(htmlNoStyles, hrefUse)

	if err := writeFileText(outputPath, htmlOut); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write HTML: %s\n%v\n", outputPath, err)
		return 1
	}

	fmt.Printf("%s %s\n", outLabelSuccess("Wrote:"), outputPath)
	fmt.Printf("%s %s\n", outLabelSuccess("Wrote:"), cssOutPath)
	fmt.Printf("- extracted style blocks: %d\n", len(cssChunks))
	fmt.Printf("- css chars: %d\n", len(cssText))
	fmt.Printf("- link href: %s\n", hrefUse)
	return 0
}

func extractStyleContentsMoveout(htmlText string) []string {
	matches := moveoutStyleBlockRe.FindAllStringSubmatch(htmlText, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

