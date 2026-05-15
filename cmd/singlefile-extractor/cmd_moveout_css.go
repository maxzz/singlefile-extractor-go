package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var moveoutStyleBlockRe = regexp.MustCompile(`(?is)<style\b[^>]*>(.*?)</style>`)

func cmdMoveoutCSS(argv []string) int {
	var (
		inputPath  string
		outputPath string
		cssOutPath string
		href       string
		showHelp   bool
	)

	fs := flag.NewFlagSet("moveout-css", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		printUsage(fs.Output(), usageSpec{
			Summary:   "Move inline <style> CSS blocks into a separate .css file and link it from the HTML.",
			UsageLine: "singlefile-extractor moveout-css --input <path> [options]",
			Options: []optionHelp{
				{Short: "i", Long: "input", Arg: "<path>", Desc: "Path to the HTML file to process. (required)"},
				{Short: "o", Long: "output", Arg: "<path>", Desc: `Where to write the updated HTML. (default: next to --input with suffix ".external-css")`},
				{Long: "css-output", Arg: "<path>", Desc: "Where to write extracted CSS. (default: <output>.css)"},
				{Long: "href", Arg: "<href>", Desc: "Optional href to use in the inserted <link> tag. (default: relative path to --css-output)"},
				{Short: "h", Long: "help", Desc: "Show help."},
			},
		})
	}

	fs.StringVar(&inputPath, "input", "", "Path to the HTML file to process. (required)")
	fs.StringVar(&inputPath, "i", "", "Path to the HTML file to process. (required)")
	fs.StringVar(&outputPath, "output", "", `Where to write the updated HTML (default: next to --input with suffix ".external-css").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the updated HTML (default: next to --input with suffix ".external-css").`)
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

	if strings.TrimSpace(inputPath) == "" {
		msg := "Missing required --input. Pass --input <path> to an HTML file."
		fmt.Fprintf(os.Stderr, "%s %s\n\n", noteLabel(), style(colors.stderr, ansiYellow, msg))
		fs.Usage()
		return 2
	}

	if outputPath == "" {
		outputPath = defaultMoveoutCSSOutputHTMLPath(inputPath)
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

	fmt.Printf("%s %s\n", wroteLabel(), outputPath)
	fmt.Printf("%s %s\n", wroteLabel(), cssOutPath)
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
