package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type extractDataURLsArgs struct {
	inputPath      string
	outputPath     string
	varsOutputPath string
	minVarURLLen   int
	varPrefix      string
	noImport       bool
	importHref     string
}

func cmdExtractDataURLs(argv []string) int {
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
		printUsage(fs.Output(), usageSpec{
			Summary:   "Extract url(data:...) occurrences into an external CSS variables file and rewrite the main CSS to reference them.",
			UsageLine: "singlefile-extractor extract-data-urls --input <path> [options]",
			Options: []optionHelp{
				{Short: "i", Long: "input", Arg: "<path>", Desc: "Path to the CSS file to process. (required)"},
				{Short: "o", Long: "output", Arg: "<path>", Desc: `Where to write the rewritten CSS. (default: next to --input with suffix "_dataurls_extracted")`},
				{Long: "vars-output", Arg: "<path>", Desc: `Where to write extracted CSS custom properties. (default: next to --output with suffix "_vars")`},
				{Long: "min-var-url-length", Arg: "<n>", Desc: "Only move existing :root custom properties into vars file if their data: URL length is >= this value. (default: 500)"},
				{Long: "var-prefix", Arg: "<prefix>", Desc: `Prefix used for generated custom properties. (default: "data-url")`},
				{Long: "no-import", Desc: "Do not insert an @import for the vars file into the rewritten CSS."},
				{Long: "import-href", Arg: "<href>", Desc: "Override the href used in the inserted @import. (default: relative path to --vars-output)"},
				{Short: "h", Long: "help", Desc: "Show help."},
			},
			OtherCommandsHeading: "Other commands",
			OtherCommands:        otherCommands("extract-data-urls"),
		})
	}

	fs.StringVar(&inputPath, "input", "", "Path to the CSS file to process. (required)")
	fs.StringVar(&inputPath, "i", "", "Path to the CSS file to process. (required)")
	fs.StringVar(&outputPath, "output", "", `Where to write the rewritten CSS (default: next to --input with suffix "_dataurls_extracted").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the rewritten CSS (default: next to --input with suffix "_dataurls_extracted").`)
	fs.StringVar(&varsOutputPath, "vars-output", "", `Where to write extracted CSS custom properties (default: next to --output with suffix "_vars").`)
	fs.IntVar(&minVarURLLen, "min-var-url-length", 500, "Only move existing :root custom properties into vars file if their data: URL length is >= this value (default: 500).")
	fs.StringVar(&varPrefix, "var-prefix", "data-url", `Prefix used for generated custom properties (default: "data-url", results in names like --data-url-... ).`)
	fs.BoolVar(&noImport, "no-import", false, "Do not insert an @import for the vars file into the rewritten CSS.")
	fs.StringVar(&importHref, "import-href", "", "Override the href used in the inserted @import (default: relative path to --vars-output).")
	fs.BoolVar(&showHelp, "help", false, "Show help.")
	fs.BoolVar(&showHelp, "h", false, "Show help.")

	if err := fs.Parse(argv); err != nil {
		fmt.Fprintln(os.Stderr, warnText(err.Error()))
		fs.Usage()
		return 2
	}
	if showHelp {
		fs.Usage()
		return 0
	}

	if strings.TrimSpace(inputPath) == "" {
		msg := "Missing required --input. Pass --input <path> to a CSS file."
		fmt.Fprintf(os.Stderr, "%s %s\n\n", noteLabel(), style(colors.stderr, ansiYellow, msg))
		fs.Usage()
		return 2
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
		fmt.Fprintln(os.Stderr, warnText(err.Error()))
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
