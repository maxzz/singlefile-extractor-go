package main

import (
	"strings"
	"testing"
)

func TestFixCSSLintErrors_RewritesLegacyIEStarHacks(t *testing.T) {
	in := "a{\n  display:inline-block;\n  *display:inline;\n  *zoom:1\n}\n"
	out := fixCSSLintErrors(in)

	if out == in {
		t.Fatalf("expected output to change")
	}
	for _, line := range strings.Split(out, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "*display:") || strings.HasPrefix(trim, "*zoom:") {
			t.Fatalf("expected legacy star hack lines removed, got line %q\nfull:\n%s", line, out)
		}
	}
	if !strings.Contains(out, "tm-error: *display:inline;") || !strings.Contains(out, "tm-error: *zoom:1") {
		t.Fatalf("expected tm-error annotations, got:\n%s", out)
	}
}

