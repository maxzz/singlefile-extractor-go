package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type dataURLHit struct {
	key      string
	urlToken string
}

type urlTokenSpan struct {
	start int
	end   int
	token string
}

type varDef struct {
	name  string
	value string
}

var (
	reSelectorTokens = regexp.MustCompile(`(#[A-Za-z0-9_-]+|\.[A-Za-z0-9_-]+|::?[A-Za-z0-9_-]+)`)
	reElementName    = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_-]*)\b`)
	reRootSelector   = regexp.MustCompile(`(?i)(^|[^\w-]):root([^\w-]|$)`)
)

func stripMatchingQuotes(s string) string {
	s2 := strings.TrimSpace(s)
	if len(s2) >= 2 && s2[0] == s2[len(s2)-1] && (s2[0] == '"' || s2[0] == '\'') {
		return s2[1 : len(s2)-1]
	}
	return s2
}

func isDataProtocol(inner string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(inner)), "data:")
}

func extractDataURLKeyAndToken(urlToken string) *dataURLHit {
	s := strings.TrimSpace(urlToken)
	if len(s) < 5 {
		return nil
	}
	// case-insensitive "url"
	if !(len(s) >= 3 && (s[0] == 'u' || s[0] == 'U') && (s[1] == 'r' || s[1] == 'R') && (s[2] == 'l' || s[2] == 'L')) {
		return nil
	}
	i := 3
	for i < len(s) && isSpaceByte(s[i]) {
		i++
	}
	if i >= len(s) || s[i] != '(' {
		return nil
	}
	if s[len(s)-1] != ')' {
		return nil
	}
	inner := strings.TrimSpace(s[i+1 : len(s)-1])
	innerUnquoted := stripMatchingQuotes(inner)
	if !isDataProtocol(innerUnquoted) {
		return nil
	}
	return &dataURLHit{
		key:      innerUnquoted,
		urlToken: s,
	}
}

func sanitizeSegment(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	lastHyphen := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteByte(c)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

func selectorHint(prelude string) string {
	s := strings.TrimSpace(prelude)
	if s == "" {
		return "global"
	}
	if strings.HasPrefix(s, "@") {
		fields := strings.Fields(strings.TrimSpace(s[1:]))
		if len(fields) == 0 {
			return "at"
		}
		h := sanitizeSegment(fields[0])
		if h == "" {
			return "at"
		}
		return h
	}

	first := strings.TrimSpace(strings.SplitN(s, ",", 2)[0])
	toks := reSelectorTokens.FindAllString(first, -1)
	if len(toks) > 0 {
		start := 0
		if len(toks) > 3 {
			start = len(toks) - 3
		}
		parts := make([]string, 0, 3)
		for _, t := range toks[start:] {
			trim := strings.TrimLeft(t, ".#:")
			if seg := sanitizeSegment(trim); seg != "" {
				parts = append(parts, seg)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "-")
		}
	}

	if m := reElementName.FindStringSubmatch(first); len(m) >= 2 {
		if seg := sanitizeSegment(m[1]); seg != "" {
			return seg
		}
	}
	return "rule"
}

func propertyHint(prop string) string {
	p := strings.ToLower(strings.TrimSpace(prop))
	if strings.HasPrefix(p, "--") {
		if seg := sanitizeSegment(p[2:]); seg != "" {
			return seg
		}
		return "var"
	}
	mapping := map[string]string{
		"background":         "bg",
		"background-image":   "bg-image",
		"mask-image":         "mask-image",
		"content":            "content",
		"src":                "src",
		"cursor":             "cursor",
		"list-style":         "list-style",
		"list-style-image":   "list-style-image",
	}
	if v, ok := mapping[p]; ok {
		return v
	}
	if seg := sanitizeSegment(p); seg != "" {
		return seg
	}
	return "prop"
}

func findTopLevelColon(statement string) int {
	var (
		inString   byte
		escape     bool
		inComment  bool
		parenDepth int
	)

	for i := 0; i < len(statement); i++ {
		c := statement[i]

		if inComment {
			if c == '*' && i+1 < len(statement) && statement[i+1] == '/' {
				i++
				inComment = false
			}
			continue
		}

		if inString != 0 {
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == inString {
				inString = 0
			}
			continue
		}

		if c == '/' && i+1 < len(statement) && statement[i+1] == '*' {
			inComment = true
			i++
			continue
		}
		if c == '"' || c == '\'' {
			inString = c
			continue
		}
		if c == '(' {
			parenDepth++
			continue
		}
		if c == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
			continue
		}
		if c == ':' && parenDepth == 0 {
			return i
		}
	}

	return -1
}

func extractPropertyName(lhs string) string {
	s := strings.TrimSpace(lhs)
	if s == "" {
		return ""
	}

	i := len(s) - 1
	for i >= 0 && isPropIdentChar(s[i]) {
		i--
	}
	name := s[i+1:]
	if name == "" {
		return ""
	}

	// Validate shape.
	if strings.HasPrefix(name, "--") {
		if len(name) < 3 {
			return ""
		}
		for j := 2; j < len(name); j++ {
			if !isPropIdentChar(name[j]) {
				return ""
			}
		}
		return name
	}
	if !isPropStartChar(name[0]) {
		return ""
	}
	for j := 1; j < len(name); j++ {
		if !isPropIdentChar(name[j]) {
			return ""
		}
	}
	return name
}

func isPropStartChar(b byte) bool {
	return isAlpha(b) || b == '_' || b == '-'
}

func isPropIdentChar(b byte) bool {
	return isAlpha(b) || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

func iterURLTokens(value string) []urlTokenSpan {
	hits := make([]urlTokenSpan, 0)

	var (
		inString  byte
		escape    bool
		inComment bool
	)

	i := 0
	n := len(value)
	for i < n {
		c := value[i]

		if inComment {
			if c == '*' && i+1 < n && value[i+1] == '/' {
				i += 2
				inComment = false
				continue
			}
			i++
			continue
		}

		if inString != 0 {
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == inString {
				inString = 0
			}
			i++
			continue
		}

		if c == '\\' {
			if i+1 < n {
				i += 2
			} else {
				i++
			}
			continue
		}
		if c == '/' && i+1 < n && value[i+1] == '*' {
			inComment = true
			i += 2
			continue
		}
		if c == '"' || c == '\'' {
			inString = c
			i++
			continue
		}

		// Look for url(...) function.
		if (c == 'u' || c == 'U') && i+2 < n &&
			(value[i+1] == 'r' || value[i+1] == 'R') &&
			(value[i+2] == 'l' || value[i+2] == 'L') {

			j := i + 3
			for j < n && isSpaceByte(value[j]) {
				j++
			}
			if j >= n || value[j] != '(' {
				i++
				continue
			}

			start := i
			j++ // after '('

			var (
				innerInString  byte
				innerEscape    bool
				innerInComment bool
			)
			for j < n {
				ch := value[j]

				if innerInComment {
					if ch == '*' && j+1 < n && value[j+1] == '/' {
						j += 2
						innerInComment = false
						continue
					}
					j++
					continue
				}
				if innerInString != 0 {
					if innerEscape {
						innerEscape = false
					} else if ch == '\\' {
						innerEscape = true
					} else if ch == innerInString {
						innerInString = 0
					}
					j++
					continue
				}

				if ch == '\\' {
					if j+1 < n {
						j += 2
					} else {
						j++
					}
					continue
				}
				if ch == '/' && j+1 < n && value[j+1] == '*' {
					innerInComment = true
					j += 2
					continue
				}
				if ch == '"' || ch == '\'' {
					innerInString = ch
					j++
					continue
				}
				if ch == ')' {
					end := j + 1
					hits = append(hits, urlTokenSpan{
						start: start,
						end:   end,
						token: value[start:end],
					})
					i = end
					break
				}
				j++
			}
			if j >= n {
				// Unterminated url(; stop.
				break
			}
			continue
		}

		i++
	}

	return hits
}

func computeImportHref(outCSS string, varsCSS string) string {
	href, err := filepath.Rel(filepath.Dir(outCSS), varsCSS)
	if err != nil {
		href = varsCSS
	}
	return filepath.ToSlash(href)
}

func splitLeadingWhitespaceAndComments(cssText string) (prefix string, rest string) {
	i := 0
	for i < len(cssText) {
		if isSpaceByte(cssText[i]) {
			i++
			continue
		}
		if i+1 < len(cssText) && cssText[i] == '/' && cssText[i+1] == '*' {
			end := strings.Index(cssText[i+2:], "*/")
			if end < 0 {
				break
			}
			i = i + 2 + end + 2
			continue
		}
		break
	}
	return cssText[:i], cssText[i:]
}

func maybeInsertImport(cssText string, href string) string {
	// Avoid duplicating an identical import.
	escaped := regexp.QuoteMeta(href)
	if regexp.MustCompile(fmt.Sprintf(`(?im)^\s*@import\s+(?:url\()?[\"']?%s[\"']?\)?\s*;`, escaped)).FindStringIndex(cssText) != nil {
		return cssText
	}

	importLine := fmt.Sprintf("@import %q;\n", href)

	prefix, rest := splitLeadingWhitespaceAndComments(cssText)
	restTrim := strings.TrimLeftFunc(rest, isSpaceRune)

	if strings.HasPrefix(strings.ToLower(restTrim), "@charset") {
		semi := strings.Index(rest, ";")
		if semi < 0 {
			return prefix + rest + "\n" + importLine
		}
		charsetStmt := rest[:semi+1]
		after := rest[semi+1:]
		if !strings.HasSuffix(charsetStmt, "\n") {
			charsetStmt += "\n"
		}
		return prefix + charsetStmt + importLine + strings.TrimLeft(after, "\n")
	}

	return prefix + importLine + strings.TrimLeft(rest, "\n")
}

type extractDataURLsResult struct {
	rewrittenCSS      string
	varsCSS           string
	extractedVars     int
	movedCustomProps  int
}

func rewriteCSSExtractDataURLs(
	cssText string,
	outCSSPath string,
	varsCSSPath string,
	minVarURLLen int,
	varPrefix string,
	noImport bool,
	importHrefOverride string,
) (extractDataURLsResult, error) {
	if minVarURLLen < 0 {
		return extractDataURLsResult{}, fmt.Errorf("--min-var-url-length must be >= 0")
	}

	varPrefixSan := sanitizeSegment(varPrefix)
	if varPrefixSan == "" {
		return extractDataURLsResult{}, fmt.Errorf("--var-prefix must contain at least one alphanumeric character")
	}

	// Map canonical data-url key -> custom property name (with leading --).
	keyToVar := map[string]string{}
	varDefs := make([]varDef, 0)
	movedCustomProps := 0
	genCounts := map[string]int{}
	preludeStack := make([]string, 0)

	currentSelectorCtx := func() string {
		for i := len(preludeStack) - 1; i >= 0; i-- {
			s := strings.TrimSpace(preludeStack[i])
			if s == "" {
				continue
			}
			if !strings.HasPrefix(s, "@") {
				return selectorHint(s)
			}
		}
		for i := len(preludeStack) - 1; i >= 0; i-- {
			s := strings.TrimSpace(preludeStack[i])
			if s == "" {
				continue
			}
			return selectorHint(s)
		}
		return "global"
	}

	inRootRule := func() bool {
		for i := len(preludeStack) - 1; i >= 0; i-- {
			s := strings.TrimSpace(preludeStack[i])
			if s == "" || strings.HasPrefix(s, "@") {
				continue
			}
			return reRootSelector.FindStringIndex(s) != nil
		}
		return false
	}

	ensureVarForURL := func(key string, urlToken string, selectorCtx string, propName string) string {
		if v, ok := keyToVar[key]; ok {
			return v
		}
		parts := make([]string, 0, 3)
		parts = append(parts, varPrefixSan)
		if seg := sanitizeSegment(selectorCtx); seg != "" {
			parts = append(parts, seg)
		}
		if seg := propertyHint(propName); seg != "" {
			parts = append(parts, seg)
		}

		base := strings.Join(parts, "-")
		base = strings.Trim(strings.ReplaceAll(strings.ReplaceAll(base, "--", "-"), "--", "-"), "-")
		for strings.Contains(base, "--") {
			base = strings.ReplaceAll(base, "--", "-")
		}
		base = strings.Trim(base, "-")
		if base == "" {
			base = varPrefixSan
		}

		genCounts[base] = genCounts[base] + 1
		varName := fmt.Sprintf("--%s-%d", base, genCounts[base])

		keyToVar[key] = varName
		varDefs = append(varDefs, varDef{name: varName, value: urlToken})
		return varName
	}

	rewriteValue := func(value string, selectorCtx string, propName string) string {
		hits := iterURLTokens(value)
		if len(hits) == 0 {
			return value
		}
		var b strings.Builder
		b.Grow(len(value))
		last := 0
		for _, h := range hits {
			b.WriteString(value[last:h.start])
			d := extractDataURLKeyAndToken(h.token)
			if d == nil {
				b.WriteString(h.token)
			} else {
				varName := ensureVarForURL(d.key, d.urlToken, selectorCtx, propName)
				b.WriteString("var(")
				b.WriteString(varName)
				b.WriteString(")")
			}
			last = h.end
		}
		b.WriteString(value[last:])
		return b.String()
	}

	maybeMoveCustomProp := func(propName string, value string) bool {
		if !strings.HasPrefix(propName, "--") {
			return false
		}
		if !inRootRule() {
			return false
		}
		valueTrimmed := strings.TrimSpace(value)
		for _, h := range iterURLTokens(value) {
			d := extractDataURLKeyAndToken(h.token)
			if d == nil {
				continue
			}
			if len(d.key) < minVarURLLen {
				continue
			}

			movedCustomProps++
			if valueTrimmed == d.urlToken {
				if _, ok := keyToVar[d.key]; !ok {
					keyToVar[d.key] = propName
				}
			}
			varDefs = append(varDefs, varDef{name: propName, value: valueTrimmed})
			return true
		}
		return false
	}

	outParts := make([]string, 0, len(cssText)/16)
	buf := make([]byte, 0, 1024)

	var (
		inString   byte
		escape     bool
		inComment  bool
		parenDepth int
	)

	flushStatement := func(stmt string, includeSemicolon bool) {
		colonI := findTopLevelColon(stmt)
		selectorCtx := currentSelectorCtx()

		if colonI < 0 {
			rewritten := rewriteValue(stmt, selectorCtx, "statement")
			if strings.TrimSpace(rewritten) != "" || includeSemicolon {
				outParts = append(outParts, rewritten)
				if includeSemicolon {
					outParts = append(outParts, ";")
				}
			}
			return
		}

		lhs := stmt[:colonI]
		rhs := stmt[colonI+1:]
		prop := extractPropertyName(lhs)
		if prop == "" {
			rewrittenRHS := rewriteValue(rhs, selectorCtx, "value")
			outParts = append(outParts, lhs, ":", rewrittenRHS)
			if includeSemicolon {
				outParts = append(outParts, ";")
			}
			return
		}

		if maybeMoveCustomProp(prop, rhs) {
			return
		}

		if strings.HasPrefix(prop, "--") {
			outParts = append(outParts, stmt)
			if includeSemicolon {
				outParts = append(outParts, ";")
			}
			return
		}

		rewrittenRHS := rewriteValue(rhs, selectorCtx, prop)
		outParts = append(outParts, lhs, ":", rewrittenRHS)
		if includeSemicolon {
			outParts = append(outParts, ";")
		}
	}

	i := 0
	n := len(cssText)
	for i < n {
		c := cssText[i]

		if inComment {
			buf = append(buf, c)
			if c == '*' && i+1 < n && cssText[i+1] == '/' {
				buf = append(buf, '/')
				i += 2
				inComment = false
				continue
			}
			i++
			continue
		}

		if inString != 0 {
			buf = append(buf, c)
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == inString {
				inString = 0
			}
			i++
			continue
		}

		if c == '\\' {
			buf = append(buf, c)
			if i+1 < n {
				buf = append(buf, cssText[i+1])
				i += 2
			} else {
				i++
			}
			continue
		}

		if c == '/' && i+1 < n && cssText[i+1] == '*' {
			buf = append(buf, '/', '*')
			i += 2
			inComment = true
			continue
		}

		if c == '"' || c == '\'' {
			buf = append(buf, c)
			inString = c
			i++
			continue
		}

		if c == '(' {
			parenDepth++
			buf = append(buf, c)
			i++
			continue
		}
		if c == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
			buf = append(buf, c)
			i++
			continue
		}

		if parenDepth == 0 && (c == ';' || c == '{' || c == '}') {
			stmt := string(buf)
			buf = buf[:0]

			switch c {
			case ';':
				flushStatement(stmt, true)
				i++
				continue
			case '{':
				outParts = append(outParts, stmt, "{")
				preludeStack = append(preludeStack, stmt)
				i++
				continue
			case '}':
				if strings.TrimSpace(stmt) != "" {
					flushStatement(stmt, false)
				}
				outParts = append(outParts, "}")
				if len(preludeStack) > 0 {
					preludeStack = preludeStack[:len(preludeStack)-1]
				}
				i++
				continue
			}
		}

		buf = append(buf, c)
		i++
	}

	if len(buf) > 0 {
		outParts = append(outParts, string(buf))
	}

	rewritten := strings.Join(outParts, "")

	if !noImport && len(varDefs) > 0 {
		href := importHrefOverride
		if href == "" {
			href = computeImportHref(outCSSPath, varsCSSPath)
		}
		rewritten = maybeInsertImport(rewritten, href)
	}

	// Vars file
	varsText := "/* No data: URLs found. */\n"
	extractedVars := 0
	if len(varDefs) > 0 {
		seen := map[string]bool{}
		ordered := make([]varDef, 0, len(varDefs))
		for _, d := range varDefs {
			if seen[d.name] {
				continue
			}
			seen[d.name] = true
			ordered = append(ordered, d)
		}
		extractedVars = len(ordered)

		lines := make([]string, 0, 3+len(ordered))
		lines = append(lines, "/* Generated by extract-data-urls */", ":root {")
		for _, d := range ordered {
			lines = append(lines, fmt.Sprintf("  %s: %s;", d.name, d.value))
		}
		lines = append(lines, "}", "")
		varsText = strings.Join(lines, "\n")
	}

	return extractDataURLsResult{
		rewrittenCSS:     rewritten,
		varsCSS:          varsText,
		extractedVars:    extractedVars,
		movedCustomProps: movedCustomProps,
	}, nil
}

