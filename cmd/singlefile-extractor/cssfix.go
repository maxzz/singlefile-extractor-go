package main

import "strings"

// fixCSSLintErrors rewrites a small set of legacy IE star-hack declarations into
// valid CSS declarations, preserving the original source line as a comment.
//
// This is intentionally conservative and only targets patterns we've seen in
// SingleFile-saved pages that cause CSS parsers/linters to error.
func fixCSSLintErrors(cssText string) string {
	if cssText == "" {
		return cssText
	}

	lines := strings.Split(cssText, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}

		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		switch {
		case strings.HasPrefix(trim, "*display:"):
			// Common pattern:
			//   display:inline-block;
			//   *display:inline;
			// Keep the modern behavior and annotate the legacy hack.
			lines[i] = indent + `display:inline-block; /* tm-error: ` + trim + ` */`
		case strings.HasPrefix(trim, "*zoom:"):
			// Keep the value but remove the invalid leading "*".
			rest := strings.TrimSpace(strings.TrimPrefix(trim, "*zoom:"))
			rest = strings.TrimSuffix(rest, ";")
			lines[i] = indent + `zoom:` + rest + `; /* tm-error: ` + trim + ` */`
		}
	}

	return strings.Join(lines, "\n")
}

