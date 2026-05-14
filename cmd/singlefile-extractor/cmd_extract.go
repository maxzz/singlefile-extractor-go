package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type iframeSrcdoc struct {
	id     string
	srcdoc string
}

type srcdocDocument struct {
	html  string
	depth int
	path  []string
}

var (
	extractStyleTagRe      = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	extractLinkStylesheet  = regexp.MustCompile(`(?i)<link\b[^>]*\brel\s*=\s*(?:"stylesheet"|'stylesheet'|stylesheet)\b[^>]*>`)
	extractBodyOpenTagRe   = regexp.MustCompile(`(?i)<body\b[^>]*>`)
	extractFormCloseTagLow = "</form>"
)

func cmdExtract(argv []string) int {
	root := repoRoot()
	defaultInput := filepath.Join(root, "tests", "Opcenter Execution (4_28_2026 3：06：53 PM).html")
	defaultOutput := filepath.Join(root, "tests", "esignature-form.html")

	var (
		inputPath  string
		outputPath string
		formID     string
		contains   string
		maxDepth   int
		showHelp   bool
	)

	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprint(os.Stdout, `Extract a <form id=...> and related inline styles from a SingleFile HTML
(nested iframe srcdoc) into a standalone HTML.

Usage:
  singlefile-extractor extract [options]

Options:
`)
		fs.PrintDefaults()
	}

	fs.StringVar(&inputPath, "input", defaultInput, "Path to the SingleFile-saved HTML file.")
	fs.StringVar(&inputPath, "i", defaultInput, "Path to the SingleFile-saved HTML file.")
	fs.StringVar(&outputPath, "output", defaultOutput, "Where to write the extracted standalone HTML.")
	fs.StringVar(&outputPath, "o", defaultOutput, "Where to write the extracted standalone HTML.")
	fs.StringVar(&formID, "form-id", "aspnetForm", "The id of the <form> element to extract (default: aspnetForm).")
	fs.StringVar(&contains, "contains", "", "Optional substring to disambiguate when multiple matching forms exist (e.g. ESigCaptureVP.aspx).")
	fs.IntVar(&maxDepth, "max-depth", 10, "Max depth to recurse through nested iframe[srcdoc] (default: 10).")
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

	outerHTML, err := readFileText(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %s\n%v\n", inputPath, err)
		return 1
	}

	candidates := iterSrcdocDocuments(outerHTML, maxDepth)
	matches := make([]srcdocDocument, 0)
	for _, d := range candidates {
		if documentContainsFormID(d.html, formID) {
			matches = append(matches, d)
		}
	}
	if contains != "" {
		filtered := matches[:0]
		for _, d := range matches {
			if strings.Contains(d.html, contains) {
				filtered = append(filtered, d)
			}
		}
		matches = filtered
	}

	if len(matches) == 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "Could not find <form id=%s> inside any nested iframe[srcdoc] documents.\n", formID)
		fmt.Fprintf(&b, "- input: %s\n", inputPath)
		fmt.Fprintf(&b, "- searched docs: %d (max_depth=%d)\n", len(candidates), maxDepth)
		if contains != "" {
			fmt.Fprintf(&b, "- contains filter: %q\n", contains)
		}
		fmt.Fprint(os.Stderr, b.String())
		return 1
	}

	targetDoc := matches[0]
	if len(matches) > 1 {
		if contains != "" {
			var b strings.Builder
			fmt.Fprintf(&b, "Found multiple nested documents containing <form id=%s> (%d matches), even after filtering by --contains.\n", formID, len(matches))
			fmt.Fprintln(&b, "Refine --contains to something more specific.\n")
			fmt.Fprintln(&b, "Matches (iframe path):")
			limit := len(matches)
			if limit > 20 {
				limit = 20
			}
			for i := 0; i < limit; i++ {
				d := matches[i]
				fmt.Fprintf(&b, "- %s (depth=%d, chars=%d)\n", strings.Join(d.path, " > "), d.depth, len(d.html))
			}
			if len(matches) > 20 {
				fmt.Fprintf(&b, "... and %d more\n", len(matches)-20)
			}
			fmt.Fprint(os.Stderr, b.String())
			return 1
		}

		targetDoc = pickBestExtractMatch(matches)
		fmt.Fprint(os.Stderr,
			"Note: multiple matching documents found; auto-selected the deepest match.\n"+
				"      Use --contains <substring> to force a different one if needed.\n"+
				fmt.Sprintf("      Selected: %s\n", strings.Join(targetDoc.path, " > ")),
		)
	}

	bodyOpen, styles, links, formHTML, err := extractFormAndStyles(targetDoc.html, formID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	outputHTML := buildOutputHTML(bodyOpen, styles, links, formHTML)

	if err := writeFileText(outputPath, outputHTML); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %s\n%v\n", outputPath, err)
		return 1
	}

	fmt.Printf("Wrote: %s\n", outputPath)
	fmt.Printf("- extracted form id: %s\n", formID)
	fmt.Printf("- source iframe path: %s\n", strings.Join(targetDoc.path, " > "))
	fmt.Printf("- styles: %d\n", len(styles))
	fmt.Printf("- link rel=stylesheet: %d\n", len(links))
	fmt.Printf("- form chars: %d\n", len(formHTML))
	return 0
}

func pickBestExtractMatch(matches []srcdocDocument) srcdocDocument {
	best := matches[0]
	bestDepth := best.depth
	bestStyles := strings.Count(strings.ToLower(best.html), "<style")
	bestLen := len(best.html)

	for _, d := range matches[1:] {
		depth := d.depth
		styles := strings.Count(strings.ToLower(d.html), "<style")
		l := len(d.html)

		better := false
		if depth != bestDepth {
			better = depth > bestDepth
		} else if styles != bestStyles {
			better = styles > bestStyles
		} else if l != bestLen {
			// Prefer smaller doc when depth/styles tie.
			better = l < bestLen
		}

		if better {
			best = d
			bestDepth = depth
			bestStyles = styles
			bestLen = l
		}
	}
	return best
}

func iterSrcdocDocuments(outerHTML string, maxDepth int) []srcdocDocument {
	docs := make([]srcdocDocument, 0)
	queue := make([]srcdocDocument, 0)

	top := parseIFrameSrcdocs(outerHTML)
	for idx, fr := range top {
		label := fr.id
		if label == "" {
			label = fmt.Sprintf("iframe#%d", idx)
		}
		queue = append(queue, srcdocDocument{
			html:  fr.srcdoc,
			depth: 1,
			path:  []string{label},
		})
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		docs = append(docs, cur)

		if cur.depth >= maxDepth {
			continue
		}

		nested := parseIFrameSrcdocs(cur.html)
		for idx, fr := range nested {
			label := fr.id
			if label == "" {
				label = fmt.Sprintf("iframe#%d", idx)
			}
			newPath := make([]string, 0, len(cur.path)+1)
			newPath = append(newPath, cur.path...)
			newPath = append(newPath, label)
			queue = append(queue, srcdocDocument{
				html:  fr.srcdoc,
				depth: cur.depth + 1,
				path:  newPath,
			})
		}
	}

	return docs
}

func parseIFrameSrcdocs(htmlText string) []iframeSrcdoc {
	out := make([]iframeSrcdoc, 0)
	i := 0
	n := len(htmlText)
	for i < n {
		if htmlText[i] != '<' {
			i++
			continue
		}
		tagText, next := parseTag(htmlText, i)
		i = next

		name := tagName(tagText)
		if name != "iframe" || isClosingTag(tagText) {
			continue
		}
		attrs := parseTagAttributes(tagText)
		srcdoc, ok := attrs["srcdoc"]
		if !ok {
			continue
		}
		out = append(out, iframeSrcdoc{
			id:     attrs["id"],
			srcdoc: srcdoc,
		})
	}
	return out
}

func documentContainsFormID(htmlText string, formID string) bool {
	target := strings.ToLower(formID)

	i := 0
	n := len(htmlText)
	for i < n {
		if htmlText[i] != '<' {
			i++
			continue
		}
		tagText, next := parseTag(htmlText, i)
		i = next

		name := tagName(tagText)
		if name != "form" || isClosingTag(tagText) {
			continue
		}
		attrs := parseTagAttributes(tagText)
		id := strings.ToLower(attrs["id"])
		if id == target {
			return true
		}
	}
	return false
}

func extractFormAndStyles(targetHTML string, formID string) (bodyOpen string, styles []string, links []string, formHTML string, err error) {
	bodyM := extractBodyOpenTagRe.FindString(targetHTML)
	if bodyM == "" {
		return "", nil, nil, "", fmt.Errorf("Could not find <body> in target srcdoc HTML.")
	}

	styles = extractStyleTagRe.FindAllString(targetHTML, -1)
	links = extractLinkStylesheet.FindAllString(targetHTML, -1)

	escaped := regexp.QuoteMeta(formID)
	formRe := regexp.MustCompile(fmt.Sprintf(
		`(?i)<form\b[^>]*\bid\s*=\s*(?:"%s"|'%s'|%s)(?=[\s>/])`,
		escaped, escaped, escaped,
	))
	loc := formRe.FindStringIndex(targetHTML)
	if loc == nil {
		return "", nil, nil, "", fmt.Errorf("Could not find <form id=%s> in target srcdoc HTML.", formID)
	}
	start := loc[0]

	lower := strings.ToLower(targetHTML)
	endRel := strings.Index(lower[loc[1]:], extractFormCloseTagLow)
	if endRel < 0 {
		return "", nil, nil, "", fmt.Errorf("Could not locate </form> end tag for %s.", formID)
	}
	end := loc[1] + endRel

	formHTML = targetHTML[start : end+len(extractFormCloseTagLow)]
	return bodyM, styles, links, formHTML, nil
}

func buildOutputHTML(bodyOpen string, styles []string, links []string, formHTML string) string {
	headBits := make([]string, 0, 3+len(styles)+len(links))
	headBits = append(headBits,
		`<meta charset="utf-8">`,
		`<meta name="viewport" content="width=device-width, initial-scale=1.0">`,
		`<title>ESignature Form</title>`,
	)
	headBits = append(headBits, links...)
	headBits = append(headBits, styles...)
	headHTML := strings.Join(headBits, "\n")

	return "<!DOCTYPE html>\n" +
		"<html lang=\"en\">\n" +
		"<head>\n" +
		headHTML + "\n" +
		"</head>\n" +
		bodyOpen + "\n" +
		formHTML + "\n" +
		"</body>\n" +
		"</html>\n"
}

