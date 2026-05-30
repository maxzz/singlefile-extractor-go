package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"singlefile-extractor-go/cmd/singlefile-extractor/colors"
	"singlefile-extractor-go/cmd/singlefile-extractor/utils"
)

func cmdFormatCSS(argv []string) int {
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
		printUsage(fs.Output(), usageSpec{
			Summary:   "Format (pretty-print) a CSS file with indentation.",
			UsageLine: "singlefile-extractor format-css --input <path> [options]",
			Options: []optionHelp{
				{Short: "i", Long: "input", Arg: "<path>", Desc: "Path to the CSS file to format. (required)"},
				{Short: "o", Long: "output", Arg: "<path>", Desc: `Where to write the formatted CSS. (default: next to --input with suffix "_formatted")`},
				{Long: "indent", Arg: "<n>", Desc: "Spaces per indent level. (default: 2)"},
				{Long: "no-extract-data-urls", Desc: "Disable automatic extraction of url(data:...) into a separate vars file."},
				{Long: "data-urls-vars-output", Arg: "<path>", Desc: `Where to write extracted data-url custom properties. (default: "<output_stem>_dataurls-vars.css")`},
				{Long: "data-urls-min-var-url-length", Arg: "<n>", Desc: "Only move existing :root custom properties into vars file when the data: URL length is >= this value. (default: 500)"},
				{Long: "data-urls-var-prefix", Arg: "<prefix>", Desc: `Prefix used for generated custom properties. (default: "data-url")`},
				{Long: "data-urls-no-import", Desc: "Do not insert an @import for the vars file into the rewritten CSS."},
				{Long: "data-urls-import-href", Arg: "<href>", Desc: "Override the href used in the inserted @import. (default: relative path to vars file)"},
				{Short: "h", Long: "help", Desc: "Show help."},
			},
			OtherCommandsHeading: "Other commands",
			OtherCommands:        otherCommands("format-css"),
			Footer: strings.TrimSpace(`
Default behavior:
- formats CSS with indentation
- by default, also extracts url(data:...) into a separate vars file and rewrites the CSS to reference them
- for data:image/... and data:font/... URLs, writes files into "assets/" and rewrites the vars to reference those files`),
		})
	}

	fs.StringVar(&inputPath, "input", "", "Path to the CSS file to format. (required)")
	fs.StringVar(&inputPath, "i", "", "Path to the CSS file to format. (required)")
	fs.StringVar(&outputPath, "output", "", `Where to write the formatted CSS (default: next to --input with suffix "_formatted").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the formatted CSS (default: next to --input with suffix "_formatted").`)
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
		fmt.Fprintln(os.Stderr, colors.WarnText(err.Error()))
		fs.Usage()
		return 2
	}
	if showHelp {
		fs.Usage()
		return 0
	}

	if strings.TrimSpace(inputPath) == "" {
		msg := "Missing required --input. Pass --input <path> to a CSS file."
		fmt.Fprintf(os.Stderr, "%s %s\n\n", colors.NoteLabel(), colors.Style(colors.Colors.Stderr, colors.AnsiYellow, msg))
		fs.Usage()
		return 2
	}

	if indentSpaces < 0 {
		fmt.Fprintln(os.Stderr, colors.WarnText("--indent must be >= 0"))
		return 2
	}

	outPath := outputPath
	if outPath == "" {
		outPath = utils.DefaultFormattedPath(inputPath)
	}

	cssText, err := utils.ReadFileText(inputPath)
	if err != nil {
		fmt.Fprint(os.Stderr, colors.WarnText(fmt.Sprintf("Failed to read input: %s\n%v\n", inputPath, err)))
		return 1
	}

	indentUnit := strings.Repeat(" ", indentSpaces)
	formatted := fixCSSLintErrors(formatCSS(cssText, indentUnit))

	if err := utils.WriteFileText(outPath, formatted); err != nil {
		fmt.Fprint(os.Stderr, colors.WarnText(fmt.Sprintf("Failed to write output: %s\n%v\n", outPath, err)))
		return 1
	}

	if noExtractDataURLs {
		fmt.Printf("%s %s\n", colors.WroteLabel(), outPath)
		fmt.Printf("- input: %s\n", inputPath)
		fmt.Printf("- indent: %d spaces\n", indentSpaces)
		fmt.Printf("- chars: %d\n", len(formatted))
		return 0
	}

	if dataURLsMinVarURLLength < 0 {
		fmt.Fprintln(os.Stderr, colors.WarnText("--data-urls-min-var-url-length must be >= 0"))
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
		fmt.Fprintln(os.Stderr, colors.WarnText(err.Error()))
		return 1
	}

	// Re-format after rewrite (extractor focuses on transformations, not formatting).
	rewritten, err := utils.ReadFileText(outPath)
	if err != nil {
		fmt.Fprint(os.Stderr, colors.WarnText(fmt.Sprintf("Failed to read rewritten CSS: %s\n%v\n", outPath, err)))
		return 1
	}
	if err := utils.WriteFileText(outPath, fixCSSLintErrors(formatCSS(rewritten, indentUnit))); err != nil {
		fmt.Fprint(os.Stderr, colors.WarnText(fmt.Sprintf("Failed to write formatted CSS: %s\n%v\n", outPath, err)))
		return 1
	}

	// Keep this script's own summary short (extractor already printed details).
	fmt.Printf("- input: %s\n", inputPath)
	fmt.Printf("- indent: %d spaces\n", indentSpaces)
	fmt.Printf("- vars: %s\n", varsOut)
	return 0
}
