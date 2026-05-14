package main

import "strings"

func collapseBlankLines(text string, maxConsecutive int) string {
	if maxConsecutive < 0 {
		return text
	}
	if text == "" {
		return text
	}

	var out strings.Builder
	out.Grow(len(text))

	blankRun := 0
	i := 0
	for i < len(text) {
		j := strings.IndexByte(text[i:], '\n')
		var line string
		if j < 0 {
			line = text[i:]
			i = len(text)
		} else {
			j = i + j
			line = text[i : j+1]
			i = j + 1
		}

		if strings.TrimSpace(line) == "" {
			blankRun++
			if blankRun > maxConsecutive {
				continue
			}
		} else {
			blankRun = 0
		}

		out.WriteString(line)
	}

	result := out.String()
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

