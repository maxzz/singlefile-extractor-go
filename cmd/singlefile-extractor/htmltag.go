package main

import (
	"html"
	"strings"
)

func parseTag(htmlText string, start int) (tagText string, nextIndex int) {
	n := len(htmlText)
	i := start + 1
	var quote byte
	for i < n {
		c := htmlText[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			i++
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			i++
			continue
		}
		if c == '>' {
			return htmlText[start : i+1], i + 1
		}
		i++
	}
	return htmlText[start:], n
}

func tagName(tagText string) string {
	s := strings.TrimSpace(tagText)
	if !strings.HasPrefix(s, "<") || len(s) < 3 {
		return ""
	}
	if strings.HasPrefix(s, "<!--") || strings.HasPrefix(s, "<!") || strings.HasPrefix(s, "<?") {
		return ""
	}

	var inner string
	if strings.HasPrefix(s, "</") {
		inner = s[2:]
	} else {
		inner = s[1:]
	}
	inner = strings.TrimLeftFunc(inner, isSpaceRune)
	if inner == "" {
		return ""
	}

	// Parse name: [A-Za-z][A-Za-z0-9:_-]*
	b := inner
	if !isAlpha(b[0]) {
		return ""
	}
	j := 1
	for j < len(b) && isTagNameChar(b[j]) {
		j++
	}
	return strings.ToLower(b[:j])
}

func isClosingTag(tagText string) bool {
	return strings.HasPrefix(strings.TrimLeftFunc(tagText, isSpaceRune), "</")
}

func isSelfClosingTag(tagText string) bool {
	return strings.HasSuffix(strings.TrimSpace(tagText), "/>")
}

func parseTagAttributes(tagText string) map[string]string {
	s := strings.TrimSpace(tagText)
	if !strings.HasPrefix(s, "<") {
		return map[string]string{}
	}
	// Drop surrounding < ... >
	s = s[1:]
	if idx := strings.LastIndexByte(s, '>'); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "/") {
		return map[string]string{}
	}
	if strings.HasSuffix(s, "/") {
		s = strings.TrimSpace(strings.TrimSuffix(s, "/"))
	}

	// Skip tag name
	i := 0
	for i < len(s) && !isSpaceByte(s[i]) {
		i++
	}

	attrs := map[string]string{}
	for i < len(s) {
		// Skip whitespace
		for i < len(s) && isSpaceByte(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '/' {
			break
		}

		// Attribute name
		start := i
		for i < len(s) && !isSpaceByte(s[i]) && s[i] != '=' && s[i] != '/' {
			i++
		}
		if i <= start {
			break
		}
		name := strings.ToLower(s[start:i])

		// Skip whitespace
		for i < len(s) && isSpaceByte(s[i]) {
			i++
		}

		// Value
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && isSpaceByte(s[i]) {
				i++
			}
			if i >= len(s) {
				attrs[name] = ""
				break
			}

			var val string
			if s[i] == '"' || s[i] == '\'' {
				q := s[i]
				i++
				startVal := i
				for i < len(s) && s[i] != q {
					i++
				}
				val = s[startVal:i]
				if i < len(s) {
					i++ // closing quote
				}
			} else {
				startVal := i
				for i < len(s) && !isSpaceByte(s[i]) && s[i] != '/' {
					i++
				}
				val = s[startVal:i]
			}
			attrs[name] = html.UnescapeString(val)
			continue
		}

		// Boolean attr / missing value
		attrs[name] = ""
	}
	return attrs
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func isSpaceRune(r rune) bool {
	// Keep this lightweight (used for trimming tag strings).
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isTagNameChar(b byte) bool {
	return isAlpha(b) || (b >= '0' && b <= '9') || b == ':' || b == '_' || b == '-'
}
