package main

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	headCloseRe = regexp.MustCompile(`(?i)</head\s*>`)
	headOpenRe  = regexp.MustCompile(`(?i)<head\b[^>]*>`)
	htmlOpenRe  = regexp.MustCompile(`(?i)<html\b[^>]*>`)
	bodyOpenRe  = regexp.MustCompile(`(?i)<body\b[^>]*>`)
)

func computeDefaultHref(outHTML string, cssOut string) string {
	href, err := filepath.Rel(filepath.Dir(outHTML), cssOut)
	if err != nil {
		href = cssOut
	}
	return filepath.ToSlash(href)
}

func hasStylesheetLink(htmlText string, href string) bool {
	targetHref := strings.TrimSpace(href)
	if targetHref == "" {
		return false
	}

	i := 0
	n := len(htmlText)
	for i < n {
		if htmlText[i] != '<' {
			i++
			continue
		}

		tagText, next := parseTag(htmlText, i)
		i = next

		if tagName(tagText) != "link" || isClosingTag(tagText) {
			continue
		}

		attrs := parseTagAttributes(tagText)
		rel := strings.TrimSpace(attrs["rel"])
		if !strings.EqualFold(rel, "stylesheet") {
			continue
		}

		hrefVal := strings.TrimSpace(attrs["href"])
		if strings.EqualFold(hrefVal, targetHref) {
			return true
		}
	}

	return false
}

func insertStylesheetLinkSimple(htmlText string, href string) string {
	if hasStylesheetLink(htmlText, href) {
		return htmlText
	}

	linkTag := fmt.Sprintf(`<link rel="stylesheet" href="%s">`, html.EscapeString(href))

	if m := headCloseRe.FindStringIndex(htmlText); m != nil {
		prefix := htmlText[:m[0]]
		suffix := htmlText[m[0]:]
		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		return prefix + linkTag + "\n" + suffix
	}

	if m := headOpenRe.FindStringIndex(htmlText); m != nil {
		insertAt := m[1]
		prefix := htmlText[:insertAt]
		suffix := htmlText[insertAt:]
		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		return prefix + linkTag + "\n" + suffix
	}

	headBlock := "<head>\n" + linkTag + "\n</head>\n"

	if m := bodyOpenRe.FindStringIndex(htmlText); m != nil {
		return htmlText[:m[0]] + headBlock + htmlText[m[0]:]
	}
	if m := htmlOpenRe.FindStringIndex(htmlText); m != nil {
		insertAt := m[1]
		prefix := htmlText[:insertAt]
		suffix := htmlText[insertAt:]
		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		return prefix + headBlock + suffix
	}
	return headBlock + htmlText
}

func insertStylesheetLinkIndented(htmlText string, href string, indentUnit string) string {
	if hasStylesheetLink(htmlText, href) {
		return htmlText
	}

	linkTag := fmt.Sprintf(`<link rel="stylesheet" href="%s">`, html.EscapeString(href))

	if m := headCloseRe.FindStringIndex(htmlText); m != nil {
		lineStart := strings.LastIndex(htmlText[:m[0]], "\n")
		if lineStart < 0 {
			lineStart = 0
		} else {
			lineStart++
		}

		closingIndent := htmlText[lineStart:m[0]]
		if strings.TrimSpace(closingIndent) != "" {
			closingIndent = ""
		}
		childIndent := closingIndent + indentUnit

		prefix := htmlText[:lineStart]
		suffix := htmlText[lineStart:]
		return prefix + childIndent + linkTag + "\n" + suffix
	}

	if m := headOpenRe.FindStringIndex(htmlText); m != nil {
		insertAt := m[1]
		prefix := htmlText[:insertAt]
		suffix := htmlText[insertAt:]

		lineStart := strings.LastIndex(htmlText[:m[0]], "\n")
		if lineStart < 0 {
			lineStart = 0
		} else {
			lineStart++
		}
		headIndent := htmlText[lineStart:m[0]]
		if strings.TrimSpace(headIndent) != "" {
			headIndent = ""
		}
		childIndent := headIndent + indentUnit

		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		suffix = strings.TrimPrefix(suffix, "\n")
		return prefix + childIndent + linkTag + "\n" + suffix
	}

	headBlock := "<head>\n" + indentUnit + linkTag + "\n</head>\n"

	if m := bodyOpenRe.FindStringIndex(htmlText); m != nil {
		return htmlText[:m[0]] + headBlock + htmlText[m[0]:]
	}
	if m := htmlOpenRe.FindStringIndex(htmlText); m != nil {
		insertAt := m[1]
		prefix := htmlText[:insertAt]
		suffix := htmlText[insertAt:]
		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		return prefix + headBlock + suffix
	}
	return headBlock + htmlText
}
