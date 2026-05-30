package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"singlefile-extractor-go/cmd/singlefile-extractor/utils"
)

func cmdFormatHTML(argv []string) int {
	var (
		inputPath               string
		outputPath              string
		indentSpaces            int
		noCSSPipeline           bool
		noExtractDataAssets     bool
		cssOutputPath           string
		cssHref                 string
		dataURLsVarsOutput      string
		dataURLsMinVarURLLength int
		dataURLsVarPrefix       string
		dataURLsNoImport        bool
		dataURLsImportHref      string
		showHelp                bool
	)

	fs := flag.NewFlagSet("format-html", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		printUsage(fs.Output(), usageSpec{
			Summary:   "Format (pretty-print) an HTML file with indentation.",
			UsageLine: "singlefile-extractor format-html --input <path> [options]",
			Options: []optionHelp{
				{Short: "i", Long: "input", Arg: "<path>", Desc: "Path to the HTML file to format. (required)"},
				{Short: "o", Long: "output", Arg: "<path>", Desc: `Where to write the formatted HTML. (default: next to --input with suffix "_formatted")`},
				{Long: "indent", Arg: "<n>", Desc: "Spaces per indent level. (default: 2)"},
				{Long: "no-css-pipeline", Desc: "Disable the default CSS pipeline (format HTML only)."},
				{Long: "no-extract-data-assets", Desc: `Disable extracting data: images/fonts from <link href> and <img src> into files under "assets/" next to the output HTML.`},
				{Long: "css-output", Arg: "<path>", Desc: `Where to write extracted CSS when <style> blocks exist. (default: "<output_stem>.css")`},
				{Long: "css-href", Arg: "<href>", Desc: "Override the href used in the inserted <link rel=stylesheet> tag. (default: relative path to --css-output)"},
				{Long: "data-urls-vars-output", Arg: "<path>", Desc: `Where to write extracted data-url custom properties. (default: "<css_stem>_dataurls-vars.css")`},
				{Long: "data-urls-min-var-url-length", Arg: "<n>", Desc: "Only move existing :root custom properties into vars file when the data: URL length is >= this value. (default: 500)"},
				{Long: "data-urls-var-prefix", Arg: "<prefix>", Desc: `Prefix used for generated custom properties. (default: "data-url")`},
				{Long: "data-urls-no-import", Desc: "Do not insert an @import for the vars file into the rewritten CSS."},
				{Long: "data-urls-import-href", Arg: "<href>", Desc: "Override the href used in the inserted @import. (default: relative path to vars file)"},
				{Short: "h", Long: "help", Desc: "Show help."},
			},
			OtherCommandsHeading: "Other commands",
			OtherCommands:        otherCommands("format-html"),
			Footer: strings.TrimSpace(`
Default CSS pipeline:
- extracts inline <style> blocks into a separate CSS file (and inserts a <link rel="stylesheet">)
- runs extract-data-urls on that CSS so url(data:...) values are moved into a vars file and referenced via var(--...)
- beautifies the rewritten CSS`),
		})
	}

	fs.StringVar(&inputPath, "input", "", "Path to the HTML file to format. (required)")
	fs.StringVar(&inputPath, "i", "", "Path to the HTML file to format. (required)")
	fs.StringVar(&outputPath, "output", "", `Where to write the formatted HTML (default: next to --input with suffix "_formatted").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the rewritten HTML (default: next to --input with suffix "_formatted").`)
	fs.IntVar(&indentSpaces, "indent", 2, "Spaces per indent level (default: 2).")
	fs.BoolVar(&noCSSPipeline, "no-css-pipeline", false, "Disable the default CSS pipeline (format HTML only).")
	fs.BoolVar(&noExtractDataAssets, "no-extract-data-assets", false, `Disable extracting data: images/fonts from <link href> and <img src> into files under "assets/" next to the output HTML.`)
	fs.StringVar(&cssOutputPath, "css-output", "", `Where to write extracted CSS when <style> blocks exist (default: "<output_stem>.css").`)
	fs.StringVar(&cssHref, "css-href", "", "Override the href used in the inserted <link rel=stylesheet> tag (default: relative path to --css-output).")
	fs.StringVar(&dataURLsVarsOutput, "data-urls-vars-output", "", `Where to write extracted data-url custom properties (default: "<css_stem>_dataurls-vars.css").`)
	fs.IntVar(&dataURLsMinVarURLLength, "data-urls-min-var-url-length", 500, "Only move existing :root custom properties into vars file when the data: URL length is >= this value (default: 500).")
	fs.StringVar(&dataURLsVarPrefix, "data-urls-var-prefix", "data-url", `Prefix used for generated custom properties (default: "data-url").`)
	fs.BoolVar(&dataURLsNoImport, "data-urls-no-import", false, "Do not insert an @import for the vars file into the rewritten CSS.")
	fs.StringVar(&dataURLsImportHref, "data-urls-import-href", "", "Override the href used in the inserted @import (default: relative path to vars file).")
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
		msg := "Missing required --input. Pass --input <path> to an HTML file."
		fmt.Fprintf(os.Stderr, "%s %s\n\n", noteLabel(), style(colors.stderr, ansiYellow, msg))
		fs.Usage()
		return 2
	}

	if indentSpaces < 0 {
		fmt.Fprintln(os.Stderr, warnText("--indent must be >= 0"))
		return 2
	}
	indentUnit := strings.Repeat(" ", indentSpaces)

	outPath := outputPath
	if outPath == "" {
		outPath = defaultFormattedPath(inputPath)
	}

	htmlText, err := utils.ReadFileText(inputPath)
	if err != nil {
		fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to read input: %s\n%v\n", inputPath, err)))
		return 1
	}

	formatted := formatHTML(htmlText, indentUnit)
	formatted = collapseBlankLines(formatted, 2)

	if noCSSPipeline {
		finalHTML := formatted
		assetsWritten := 0
		if !noExtractDataAssets {
			var replaced int
			finalHTML, assetsWritten, replaced, err = extractDataAssetsFromHTML(finalHTML, outPath)
			if err != nil {
				fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to extract data assets: %v\n", err)))
				return 1
			}
			_ = replaced
		}

		if err := utils.WriteFileText(outPath, finalHTML); err != nil {
			fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to write output: %s\n%v\n", outPath, err)))
			return 1
		}
		fmt.Printf("%s %s\n", wroteLabel(), outPath)
		fmt.Printf("- input: %s\n", inputPath)
		fmt.Printf("- indent: %d spaces\n", indentSpaces)
		fmt.Printf("- chars: %d\n", len(finalHTML))
		if assetsWritten > 0 {
			fmt.Printf("- data assets written: %d\n", assetsWritten)
		}
		return 0
	}

	cssFiles := make([]string, 0)
	varsFiles := make([]string, 0)
	finalHTML := ""
	assetsWritten := 0

	styleChunks := extractStyleContentsFormattedHTML(formatted)
	if len(styleChunks) > 0 {
		cssOut := cssOutputPath
		if cssOut == "" {
			cssOut = replaceExt(outPath, ".css")
		}

		cssText := strings.Join(styleChunks, "\n\n")
		cssText = strings.TrimRight(cssText, "\r\n") + "\n"
		if err := utils.WriteFileText(cssOut, cssText); err != nil {
			fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to write CSS: %s\n%v\n", cssOut, err)))
			return 1
		}

		href := cssHref
		if href == "" {
			href = computeDefaultHref(outPath, cssOut)
		}

		htmlNoStyles := removeStyleBlocksFormattedHTML(formatted)
		htmlLinked := insertStylesheetLinkIndented(htmlNoStyles, href, indentUnit)
		htmlLinked = collapseBlankLines(htmlLinked, 2)
		finalHTML = htmlLinked

		cssFiles = append(cssFiles, cssOut)
	} else {
		finalHTML = formatted

		for _, href := range iterLinkedStylesheetHrefs(formatted) {
			h := strings.TrimSpace(href)
			if h == "" {
				continue
			}
			hl := strings.ToLower(h)
			if strings.HasPrefix(hl, "http://") || strings.HasPrefix(hl, "https://") || strings.HasPrefix(h, "//") || strings.HasPrefix(hl, "data:") {
				continue
			}

			cssPath := h
			if !filepath.IsAbs(cssPath) {
				cssPath = filepath.Clean(filepath.Join(filepath.Dir(outPath), cssPath))
			}
			if info, err := os.Stat(cssPath); err == nil && !info.IsDir() {
				cssFiles = append(cssFiles, cssPath)
			}
		}
	}

	if len(cssFiles) > 0 {
		if dataURLsMinVarURLLength < 0 {
			fmt.Fprintln(os.Stderr, warnText("--data-urls-min-var-url-length must be >= 0"))
			return 2
		}
		if dataURLsVarsOutput != "" && len(cssFiles) > 1 {
			fmt.Fprintln(os.Stderr, warnText("--data-urls-vars-output can only be used when processing a single CSS file."))
			return 2
		}

		for _, cssPath := range cssFiles {
			varsOut := dataURLsVarsOutput
			if varsOut == "" {
				ext := filepath.Ext(cssPath)
				stem := strings.TrimSuffix(filepath.Base(cssPath), ext)
				varsOut = filepath.Join(filepath.Dir(cssPath), stem+"_dataurls-vars"+ext)
			}

			if err := runExtractDataURLs(extractDataURLsArgs{
				inputPath:      cssPath,
				outputPath:     cssPath, // rewrite in-place
				varsOutputPath: varsOut,
				minVarURLLen:   dataURLsMinVarURLLength,
				varPrefix:      dataURLsVarPrefix,
				noImport:       dataURLsNoImport,
				importHref:     dataURLsImportHref,
			}); err != nil {
				fmt.Fprintln(os.Stderr, warnText(err.Error()))
				return 1
			}
			varsFiles = append(varsFiles, varsOut)
		}

		// Beautify rewritten CSS (extract-data-urls focuses on transformations, not formatting).
		cssIndent := strings.Repeat(" ", indentSpaces)
		for _, cssPath := range cssFiles {
			cssText, err := utils.ReadFileText(cssPath)
			if err != nil {
				fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to read CSS: %s\n%v\n", cssPath, err)))
				return 1
			}
			formattedCSS := formatCSS(cssText, cssIndent)
			formattedCSS = fixCSSLintErrors(formattedCSS)
			if err := utils.WriteFileText(cssPath, formattedCSS); err != nil {
				fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to write CSS: %s\n%v\n", cssPath, err)))
				return 1
			}
		}
	}

	if !noExtractDataAssets {
		var replaced int
		finalHTML, assetsWritten, replaced, err = extractDataAssetsFromHTML(finalHTML, outPath)
		if err != nil {
			fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to extract data assets: %v\n", err)))
			return 1
		}
		_ = replaced
	}

	if err := utils.WriteFileText(outPath, finalHTML); err != nil {
		fmt.Fprint(os.Stderr, warnText(fmt.Sprintf("Failed to write HTML: %s\n%v\n", outPath, err)))
		return 1
	}

	fmt.Printf("%s %s\n", wroteLabel(), outPath)
	fmt.Printf("- input: %s\n", inputPath)
	fmt.Printf("- indent: %d spaces\n", indentSpaces)
	if len(cssFiles) > 0 {
		fmt.Printf("- css files processed: %d\n", len(cssFiles))
	}
	if len(varsFiles) > 0 {
		fmt.Printf("- vars files written: %d\n", len(varsFiles))
	}
	if assetsWritten > 0 {
		fmt.Printf("- data assets written: %d\n", assetsWritten)
	}
	return 0
}
