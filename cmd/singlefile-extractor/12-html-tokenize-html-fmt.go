package main

import (
	"regexp"
	"strings"
)

type htmlToken struct {
	kind  string // "tag" | "text"
	value string
}

var (
	voidElements = map[string]bool{
		"area":   true,
		"base":   true,
		"br":     true,
		"col":    true,
		"embed":  true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"link":   true,
		"meta":   true,
		"param":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}

	rawtextElements = map[string]bool{
		"script":   true,
		"style":    true,
		"textarea": true,
		"pre":      true,
	}

	impliedCloseOnOpen = map[string]map[string]bool{
		"li":       {"li": true},
		"dt":       {"dt": true, "dd": true},
		"dd":       {"dt": true, "dd": true},
		"option":   {"option": true},
		"optgroup": {"option": true},
		"td":       {"td": true, "th": true},
		"th":       {"td": true, "th": true},
		"tr":       {"td": true, "th": true, "tr": true},
		"thead":    {"td": true, "th": true, "tr": true, "thead": true, "tbody": true, "tfoot": true},
		"tbody":    {"td": true, "th": true, "tr": true, "thead": true, "tbody": true, "tfoot": true},
		"tfoot":    {"td": true, "th": true, "tr": true, "thead": true, "tbody": true, "tfoot": true},
	}

	fmtStyleBlockRe        = regexp.MustCompile(`(?ims)^[ \t]*<style\b[^>]*>(.*?)</style>[ \t]*\r?\n?`)
	fmtLinkStylesheetTagRe = regexp.MustCompile(`(?i)<link\b[^>]*\brel\s*=\s*(?:"stylesheet"|'stylesheet'|stylesheet)\b[^>]*>`)
	fmtHrefAttrRe          = regexp.MustCompile(`(?i)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
)

func tokenizeHTML(htmlText string) []htmlToken {
	tokens := make([]htmlToken, 0, len(htmlText)/32)
	i := 0
	n := len(htmlText)

	for i < n {
		if htmlText[i] != '<' {
			j := strings.IndexByte(htmlText[i:], '<')
			if j < 0 {
				j = n - i
			}
			tokens = append(tokens, htmlToken{kind: "text", value: htmlText[i : i+j]})
			i = i + j
			continue
		}

		// Comment
		if strings.HasPrefix(htmlText[i:], "<!--") {
			j := strings.Index(htmlText[i+4:], "-->")
			if j < 0 {
				tokens = append(tokens, htmlToken{kind: "tag", value: htmlText[i:]})
				break
			}
			end := i + 4 + j + 3
			tokens = append(tokens, htmlToken{kind: "tag", value: htmlText[i:end]})
			i = end
			continue
		}

		tagText, next := parseTag(htmlText, i)
		tokens = append(tokens, htmlToken{kind: "tag", value: tagText})
		i = next

		// If this is an opening rawtext element, don't try to tokenize its contents.
		name := tagName(tagText)
		if name == "" {
			continue
		}
		if isClosingTag(tagText) {
			continue
		}
		if isSelfClosingTag(tagText) || voidElements[name] {
			continue
		}
		if !rawtextElements[name] {
			continue
		}

		closeRe := regexp.MustCompile(`(?i)</\s*` + regexp.QuoteMeta(name) + `\s*>`)
		loc := closeRe.FindStringIndex(htmlText[i:])
		if loc == nil {
			tokens = append(tokens, htmlToken{kind: "text", value: htmlText[i:]})
			break
		}
		closeStart := i + loc[0]
		closeEnd := i + loc[1]
		tokens = append(tokens, htmlToken{kind: "text", value: htmlText[i:closeStart]})
		tokens = append(tokens, htmlToken{kind: "tag", value: htmlText[closeStart:closeEnd]})
		i = closeEnd
	}

	return tokens
}

func popThroughNearest(stack []string, names map[string]bool) []string {
	for idx := len(stack) - 1; idx >= 0; idx-- {
		if names[stack[idx]] {
			return stack[:idx]
		}
	}
	return stack
}

func popCloseTag(stack []string, name string) (newStack []string, indentLevel int) {
	for idx := len(stack) - 1; idx >= 0; idx-- {
		if stack[idx] == name {
			return stack[:idx], idx
		}
	}
	return stack, len(stack)
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func formatHTML(htmlText string, indentUnit string) string {
	tokens := tokenizeHTML(htmlText)

	lines := make([]string, 0, len(tokens))
	stack := make([]string, 0, 64)

	emit := func(line string) {
		if line == "" {
			return
		}
		lines = append(lines, line)
	}

	for _, tok := range tokens {
		if tok.kind == "text" {
			if strings.TrimSpace(tok.value) == "" {
				continue
			}
			for _, rawLine := range splitLines(tok.value) {
				s := strings.TrimSpace(rawLine)
				if s == "" {
					continue
				}
				emit(strings.Repeat(indentUnit, len(stack)) + s)
			}
			continue
		}

		tag := strings.TrimSpace(tok.value)
		if tag == "" {
			continue
		}

		name := tagName(tag)
		if isClosingTag(tag) {
			indentLevel := len(stack)
			if name != "" {
				var newStack []string
				newStack, indentLevel = popCloseTag(stack, name)
				stack = newStack
			}
			emit(strings.Repeat(indentUnit, indentLevel) + tag)
			continue
		}

		if name == "" {
			emit(strings.Repeat(indentUnit, len(stack)) + tag)
			continue
		}

		if implied, ok := impliedCloseOnOpen[name]; ok {
			stack = popThroughNearest(stack, implied)
		}

		emit(strings.Repeat(indentUnit, len(stack)) + tag)

		if isSelfClosingTag(tag) || voidElements[name] {
			continue
		}
		stack = append(stack, name)
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\r\n") + "\n"
}

func extractStyleContentsFormattedHTML(htmlText string) []string {
	matches := fmtStyleBlockRe.FindAllStringSubmatch(htmlText, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

func removeStyleBlocksFormattedHTML(htmlText string) string {
	return fmtStyleBlockRe.ReplaceAllString(htmlText, "")
}

func iterLinkedStylesheetHrefs(htmlText string) []string {
	tags := fmtLinkStylesheetTagRe.FindAllString(htmlText, -1)
	hrefs := make([]string, 0, len(tags))
	for _, tag := range tags {
		m := fmtHrefAttrRe.FindStringSubmatch(tag)
		if len(m) < 4 {
			continue
		}
		href := strings.TrimSpace(firstNonEmpty(m[1], m[2], m[3]))
		if href != "" {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
