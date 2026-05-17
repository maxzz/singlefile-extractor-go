package main

import (
	"html"
	"strings"
)

func replaceTagAttrValue(tagText string, attrName string, newValue string) (string, bool) {
	if !strings.HasPrefix(tagText, "<") {
		return tagText, false
	}

	target := strings.ToLower(strings.TrimSpace(attrName))
	if target == "" {
		return tagText, false
	}

	i := 1
	n := len(tagText)

	// Skip whitespace after '<'
	for i < n && isSpaceByte(tagText[i]) {
		i++
	}
	if i < n && tagText[i] == '/' {
		return tagText, false
	}

	// Skip tag name.
	for i < n && !isSpaceByte(tagText[i]) && tagText[i] != '>' {
		i++
	}

	for i < n {
		// Skip whitespace.
		for i < n && isSpaceByte(tagText[i]) {
			i++
		}
		if i >= n {
			break
		}
		if tagText[i] == '>' {
			break
		}
		if tagText[i] == '/' {
			i++
			continue
		}

		// Attribute name.
		nameStart := i
		for i < n && !isSpaceByte(tagText[i]) && tagText[i] != '=' && tagText[i] != '>' && tagText[i] != '/' {
			i++
		}
		if i <= nameStart {
			break
		}
		name := strings.ToLower(tagText[nameStart:i])

		// Skip whitespace.
		for i < n && isSpaceByte(tagText[i]) {
			i++
		}

		if i < n && tagText[i] == '=' {
			i++ // '='
			for i < n && isSpaceByte(tagText[i]) {
				i++
			}
			if i >= n {
				break
			}

			valStart := i
			valEnd := i

			if tagText[i] == '"' || tagText[i] == '\'' {
				q := tagText[i]
				i++
				for i < n && tagText[i] != q {
					i++
				}
				if i < n {
					i++ // include closing quote
				}
				valEnd = i
			} else {
				for i < n && !isSpaceByte(tagText[i]) && tagText[i] != '>' {
					// Treat "/>" as the end of the tag, not part of the value.
					if tagText[i] == '/' && i+1 < n && tagText[i+1] == '>' {
						break
					}
					i++
				}
				valEnd = i
			}

			if name == target {
				escaped := html.EscapeString(newValue)
				repl := `"` + escaped + `"`
				return tagText[:valStart] + repl + tagText[valEnd:], true
			}
			continue
		}
	}

	return tagText, false
}

