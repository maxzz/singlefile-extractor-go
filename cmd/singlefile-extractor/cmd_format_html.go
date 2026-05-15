package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdFormatHTML(argv []string) int {
	var (
		inputPath               string
		outputPath              string
		indentSpaces            int
		noCSSPipeline           bool
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
		fmt.Fprint(os.Stdout, `Format (pretty-print) an HTML file with indentation.

By default it also runs a CSS pipeline:
- extracts inline <style> blocks into a separate CSS file (and inserts a <link rel="stylesheet">)
- runs extract-data-urls on that CSS so url(data:...) values are moved into a vars file and referenced via var(--...)
- beautifies the rewritten CSS

Usage:
  singlefile-extractor format-html [options]

Options:
`)
		fs.PrintDefaults()
	}

	fs.StringVar(&inputPath, "input", "", "Path to the HTML file to format. (required)")
	fs.StringVar(&inputPath, "i", "", "Path to the HTML file to format. (required)")
	fs.StringVar(&outputPath, "output", "", `Where to write the formatted HTML (default: "<input>_formatted.html").`)
	fs.StringVar(&outputPath, "o", "", `Where to write the formatted HTML (default: "<input>_formatted.html").`)
	fs.IntVar(&indentSpaces, "indent", 2, "Spaces per indent level (default: 2).")
	fs.BoolVar(&noCSSPipeline, "no-css-pipeline", false, "Disable the default CSS pipeline (format HTML only).")
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

	if indentSpaces < 0 {
		fmt.Fprintln(os.Stderr, "--indent must be >= 0")
		return 2
	}
	indentUnit := strings.Repeat(" ", indentSpaces)

	outPath := outputPath
	if outPath == "" {
		outPath = defaultFormattedPath(inputPath)
	}

	htmlText, err := readFileText(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n%v\n", inputPath, err)
		return 1
	}

	formatted := formatHTML(htmlText, indentUnit)
	formatted = collapseBlankLines(formatted, 2)

	if noCSSPipeline {
		if err := writeFileText(outPath, formatted); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write output: %s\n%v\n", outPath, err)
			return 1
		}
		fmt.Printf("%s %s\n", wroteLabel(), outPath)
		fmt.Printf("- input: %s\n", inputPath)
		fmt.Printf("- indent: %d spaces\n", indentSpaces)
		fmt.Printf("- chars: %d\n", len(formatted))
		return 0
	}

	cssFiles := make([]string, 0)
	varsFiles := make([]string, 0)

	styleChunks := extractStyleContentsFormattedHTML(formatted)
	if len(styleChunks) > 0 {
		cssOut := cssOutputPath
		if cssOut == "" {
			cssOut = replaceExt(outPath, ".css")
		}

		cssText := strings.Join(styleChunks, "\n\n")
		cssText = strings.TrimRight(cssText, "\r\n") + "\n"
		if err := writeFileText(cssOut, cssText); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write CSS: %s\n%v\n", cssOut, err)
			return 1
		}

		href := cssHref
		if href == "" {
			href = computeDefaultHref(outPath, cssOut)
		}

		htmlNoStyles := removeStyleBlocksFormattedHTML(formatted)
		htmlLinked := insertStylesheetLinkIndented(htmlNoStyles, href, indentUnit)
		htmlLinked = collapseBlankLines(htmlLinked, 2)

		if err := writeFileText(outPath, htmlLinked); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write HTML: %s\n%v\n", outPath, err)
			return 1
		}

		cssFiles = append(cssFiles, cssOut)
	} else {
		if err := writeFileText(outPath, formatted); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write HTML: %s\n%v\n", outPath, err)
			return 1
		}

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
			fmt.Fprintln(os.Stderr, "--data-urls-min-var-url-length must be >= 0")
			return 2
		}
		if dataURLsVarsOutput != "" && len(cssFiles) > 1 {
			fmt.Fprintln(os.Stderr, "--data-urls-vars-output can only be used when processing a single CSS file.")
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
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			varsFiles = append(varsFiles, varsOut)
		}

		// Beautify rewritten CSS (extract-data-urls focuses on transformations, not formatting).
		cssIndent := strings.Repeat(" ", indentSpaces)
		for _, cssPath := range cssFiles {
			cssText, err := readFileText(cssPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read CSS: %s\n%v\n", cssPath, err)
				return 1
			}
			if err := writeFileText(cssPath, formatCSS(cssText, cssIndent)); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write CSS: %s\n%v\n", cssPath, err)
				return 1
			}
		}
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
	return 0
}
