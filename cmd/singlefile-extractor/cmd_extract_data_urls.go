package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type extractDataURLsArgs struct {
	inputPath         string
	outputPath        string
	varsOutputPath    string
	minVarURLLen      int
	varPrefix         string
	noImport          bool
	importHref        string
}

func cmdExtractDataURLs(argv []string) int {
	root := repoRoot()
	defaultInput := filepath.Join(root, "tests-local", "esig.smoke_formatted.css")

	var (
		inputPath      string
		outputPath     string
		varsOutputPath string
		minVarURLLen   int
		varPrefix      string
		noImport       bool
		importHref     string
		showHelp       bool
	)

	fs := flag.NewFlagSet("extract-data-urls", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprint(os.Stdout, `Extract url(data:...) occurrences into an external CSS variables file and rewrite the main CSS to reference them.

Usage:
  singlefile-extractor extract-data-urls [options]

Options:
`)
		fs.PrintDefaults()
	}

	fs.StringVar(&inputPath, "input", defaultInput, "Path to the CSS file to process.")
	fs.StringVar(&inputPath, "i", defaultInput, "Path to the CSS file to process.")
	fs.StringVar(&outputPath, "output", "", `Where to write the rewritten CSS (default: "<input>_dataurls_extracted.css").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the rewritten CSS (default: "<input>_dataurls_extracted.css").`)
	fs.StringVar(&varsOutputPath, "vars-output", "", `Where to write extracted CSS custom properties (default: "<output>_vars.css").`)
	fs.IntVar(&minVarURLLen, "min-var-url-length", 500, "Only move existing :root custom properties into vars file if their data: URL length is >= this value (default: 500).")
	fs.StringVar(&varPrefix, "var-prefix", "data-url", `Prefix used for generated custom properties (default: "data-url", results in names like --data-url-... ).`)
	fs.BoolVar(&noImport, "no-import", false, "Do not insert an @import for the vars file into the rewritten CSS.")
	fs.StringVar(&importHref, "import-href", "", "Override the href used in the inserted @import (default: relative path to --vars-output).")
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

	outCSS := outputPath
	if outCSS == "" {
		ext := filepath.Ext(inputPath)
		stem := strings.TrimSuffix(filepath.Base(inputPath), ext)
		outCSS = filepath.Join(filepath.Dir(inputPath), stem+"_dataurls_extracted"+ext)
	}
	varsCSS := varsOutputPath
	if varsCSS == "" {
		ext := filepath.Ext(outCSS)
		stem := strings.TrimSuffix(filepath.Base(outCSS), ext)
		varsCSS = filepath.Join(filepath.Dir(outCSS), stem+"_vars"+ext)
	}

	args := extractDataURLsArgs{
		inputPath:      inputPath,
		outputPath:     outCSS,
		varsOutputPath: varsCSS,
		minVarURLLen:   minVarURLLen,
		varPrefix:      varPrefix,
		noImport:       noImport,
		importHref:     importHref,
	}

	if err := runExtractDataURLs(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runExtractDataURLs(args extractDataURLsArgs) error {
	cssText, err := readFileText(args.inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input: %s\n%w", args.inputPath, err)
	}

	res, err := rewriteCSSExtractDataURLs(
		cssText,
		args.outputPath,
		args.varsOutputPath,
		args.minVarURLLen,
		args.varPrefix,
		args.noImport,
		args.importHref,
	)
	if err != nil {
		return err
	}

	if err := writeFileText(args.outputPath, res.rewrittenCSS); err != nil {
		return fmt.Errorf("failed to write: %s\n%w", args.outputPath, err)
	}
	if err := writeFileText(args.varsOutputPath, res.varsCSS); err != nil {
		return fmt.Errorf("failed to write: %s\n%w", args.varsOutputPath, err)
	}

	fmt.Printf("%s %s\n", wroteLabel(), args.outputPath)
	fmt.Printf("%s %s\n", wroteLabel(), args.varsOutputPath)
	fmt.Printf("- extracted vars: %d\n", res.extractedVars)
	fmt.Printf("- moved root custom props (min_len=%d): %d\n", args.minVarURLLen, res.movedCustomProps)
	return nil
}

