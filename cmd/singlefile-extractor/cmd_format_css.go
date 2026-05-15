package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdFormatCSS(argv []string) int {
	root := repoRoot()
	defaultInput := filepath.Join(root, "tests", "esignature-form.css")

	var (
		inputPath               string
		outputPath              string
		indentSpaces            int
		noExtractDataURLs       bool
		dataURLsVarsOutput      string
		dataURLsMinVarURLLength int
		dataURLsVarPrefix       string
		dataURLsNoImport        bool
		dataURLsImportHref      string
		showHelp                bool
	)

	fs := flag.NewFlagSet("format-css", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprint(os.Stdout, `Format (pretty-print) a CSS file with indentation.

By default it also extracts url(data:...) into a separate vars file and rewrites the CSS to reference them.

Usage:
  singlefile-extractor format-css [options]

Options:
`)
		fs.PrintDefaults()
	}

	fs.StringVar(&inputPath, "input", defaultInput, "Path to the CSS file to format.")
	fs.StringVar(&inputPath, "i", defaultInput, "Path to the CSS file to format.")
	fs.StringVar(&outputPath, "output", "", `Where to write the formatted CSS (default: "<input>_formatted.css").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the formatted CSS (default: "<input>_formatted.css").`)
	fs.IntVar(&indentSpaces, "indent", 2, "Spaces per indent level (default: 2).")
	fs.BoolVar(&noExtractDataURLs, "no-extract-data-urls", false, "Disable automatic extraction of url(data:...) into a separate vars file.")
	fs.StringVar(&dataURLsVarsOutput, "data-urls-vars-output", "", `Where to write extracted data-url custom properties (default: "<output_stem>_dataurls-vars.css").`)
	fs.IntVar(&dataURLsMinVarURLLength, "data-urls-min-var-url-length", 500, "Only move existing :root custom properties into vars file when the data: URL length is >= this value (default: 500).")
	fs.StringVar(&dataURLsVarPrefix, "data-urls-var-prefix", "data-url", `Prefix used for generated custom properties (default: "data-url").`)
	fs.BoolVar(&dataURLsNoImport, "data-urls-no-import", false, "Do not insert an @import for the vars file into the rewritten CSS.")
	fs.StringVar(&dataURLsImportHref, "data-urls-import-href", "", "Override the href used in the inserted @import (default: relative path to vars file).")
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

	if indentSpaces < 0 {
		fmt.Fprintln(os.Stderr, "--indent must be >= 0")
		return 2
	}

	outPath := outputPath
	if outPath == "" {
		outPath = defaultFormattedPath(inputPath)
	}

	cssText, err := readFileText(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n%v\n", inputPath, err)
		return 1
	}

	indentUnit := strings.Repeat(" ", indentSpaces)
	formatted := formatCSS(cssText, indentUnit)

	if err := writeFileText(outPath, formatted); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %s\n%v\n", outPath, err)
		return 1
	}

	if noExtractDataURLs {
		fmt.Printf("Wrote: %s\n", outPath)
		fmt.Printf("- input: %s\n", inputPath)
		fmt.Printf("- indent: %d spaces\n", indentSpaces)
		fmt.Printf("- chars: %d\n", len(formatted))
		return 0
	}

	if dataURLsMinVarURLLength < 0 {
		fmt.Fprintln(os.Stderr, "--data-urls-min-var-url-length must be >= 0")
		return 2
	}

	varsOut := dataURLsVarsOutput
	if varsOut == "" {
		ext := filepath.Ext(outPath)
		stem := strings.TrimSuffix(filepath.Base(outPath), ext)
		varsOut = filepath.Join(filepath.Dir(outPath), stem+"_dataurls-vars"+ext)
	}

	// Rewrite the formatted CSS in-place (extractor prints its own summary).
	if err := runExtractDataURLs(extractDataURLsArgs{
		inputPath:      outPath,
		outputPath:     outPath,
		varsOutputPath: varsOut,
		minVarURLLen:   dataURLsMinVarURLLength,
		varPrefix:      dataURLsVarPrefix,
		noImport:       dataURLsNoImport,
		importHref:     dataURLsImportHref,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Re-format after rewrite (extractor focuses on transformations, not formatting).
	rewritten, err := readFileText(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read rewritten CSS: %s\n%v\n", outPath, err)
		return 1
	}
	if err := writeFileText(outPath, formatCSS(rewritten, indentUnit)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write formatted CSS: %s\n%v\n", outPath, err)
		return 1
	}

	// Keep this script's own summary short (extractor already printed details).
	fmt.Printf("- input: %s\n", inputPath)
	fmt.Printf("- indent: %d spaces\n", indentSpaces)
	fmt.Printf("- vars: %s\n", varsOut)
	return 0
}

