package main

import (
	"strings"
	"unicode"
)

func removeEmptyRuleSets(cssText string) string {
	if cssText == "" {
		return cssText
	}

	// Match Python's splitlines() behavior (no trailing empty line).
	lines := strings.Split(cssText, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return cssText
	}

	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.HasSuffix(strings.TrimSpace(line), "{") {
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j < len(lines) && strings.TrimSpace(lines[j]) == "}" {
				// Skip the whole empty block, plus any following blank lines.
				i = j + 1
				for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
					i++
				}
				continue
			}
		}
		out = append(out, line)
		i++
	}

	if len(out) == 0 {
		return ""
	}
	joined := strings.Join(out, "\n")
	joined = strings.TrimRightFunc(joined, unicode.IsSpace)
	return joined + "\n"
}

func formatCSS(cssText string, indent string) string {
	out := make([]byte, 0, len(cssText)+len(cssText)/4)

	level := 0
	parenDepth := 0

	var inString byte
	escape := false
	inComment := false

	pendingSpace := false
	atLineStart := true

	indentBytes := []byte(indent)

	appendByte := func(ch byte) {
		if atLineStart && ch != '\n' {
			for i := 0; i < level; i++ {
				out = append(out, indentBytes...)
			}
			atLineStart = false
		}
		out = append(out, ch)
		if ch == '\n' {
			atLineStart = true
		}
	}

	appendString := func(s string) {
		for i := 0; i < len(s); i++ {
			appendByte(s[i])
		}
	}

	trimTrailingSpaces := func() {
		for len(out) > 0 && out[len(out)-1] == ' ' {
			out = out[:len(out)-1]
		}
	}

	newline := func() {
		trimTrailingSpaces()
		if len(out) == 0 || out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		atLineStart = true
		pendingSpace = false
	}

	i := 0
	n := len(cssText)
	for i < n {
		c := cssText[i]

		if inComment {
			if c == '*' && i+1 < n && cssText[i+1] == '/' {
				appendString("*/")
				i += 2
				inComment = false
				pendingSpace = true
				continue
			}
			appendByte(c)
			i++
			continue
		}

		if inString != 0 {
			appendByte(c)
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

		// Not in comment or string.
		if c == '\\' {
			if pendingSpace && !atLineStart {
				appendByte(' ')
			}
			pendingSpace = false
			appendByte('\\')
			if i+1 < n {
				appendByte(cssText[i+1])
				i += 2
			} else {
				i++
			}
			continue
		}

		if c == '/' && i+1 < n && cssText[i+1] == '*' {
			if pendingSpace && !atLineStart {
				appendByte(' ')
			}
			pendingSpace = false
			appendString("/*")
			i += 2
			inComment = true
			continue
		}

		if c == '"' || c == '\'' {
			if pendingSpace && !atLineStart {
				appendByte(' ')
			}
			pendingSpace = false
			appendByte(c)
			inString = c
			i++
			continue
		}

		if isSpaceByte(c) {
			pendingSpace = true
			i++
			continue
		}

		if c == '(' {
			if pendingSpace && !atLineStart {
				appendByte(' ')
			}
			pendingSpace = false
			appendByte('(')
			parenDepth++
			i++
			continue
		}

		if c == ')' {
			pendingSpace = false
			appendByte(')')
			if parenDepth > 0 {
				parenDepth--
			}
			i++
			continue
		}

		if c == '{' {
			if pendingSpace && !atLineStart {
				appendByte(' ')
			}
			pendingSpace = false

			if len(out) > 0 && out[len(out)-1] != ' ' && out[len(out)-1] != '\n' {
				appendByte(' ')
			}
			appendByte('{')
			newline()
			level++
			i++
			continue
		}

		if c == '}' {
			pendingSpace = false
			if level > 0 {
				level--
			}
			if !atLineStart {
				newline()
			}
			appendByte('}')
			newline()
			i++
			continue
		}

		if c == ';' {
			pendingSpace = false
			appendByte(';')
			if parenDepth == 0 {
				newline()
			}
			i++
			continue
		}

		if c == ',' {
			if pendingSpace && !atLineStart {
				appendByte(' ')
			}
			pendingSpace = false
			appendByte(',')
			if parenDepth == 0 {
				pendingSpace = true
			}
			i++
			continue
		}

		// Default character.
		if pendingSpace && !atLineStart {
			appendByte(' ')
		}
		pendingSpace = false
		appendByte(c)
		i++
	}

	newline()
	return removeEmptyRuleSets(string(out))
}
