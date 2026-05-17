package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// format-html keeps <!-- ... --> comments in the output. Verified on tests/extra_field.html (the SingleFile header comment is still present in tests/extra_field_formatted.html):
// I also added a small regression test in cmd/singlefile-extractor/htmlfmt_test.go to prevent this from breaking later.

func TestFormatHTML_PreservesHTMLComments(t *testing.T) {
	in := "<!DOCTYPE html> <html><!--\n Page saved with SingleFile \n--><head><title>x</title></head></html>"
	out := formatHTML(in, "  ")

	if !strings.Contains(out, "<!--") {
		t.Fatalf("expected output to contain <!--, got:\n%s", out)
	}
	if !strings.Contains(out, "Page saved with SingleFile") {
		t.Fatalf("expected output to contain comment body, got:\n%s", out)
	}
	if !strings.Contains(out, "-->") {
		t.Fatalf("expected output to contain -->, got:\n%s", out)
	}
}

func TestExtractDataAssetsFromHTML_PreservesHTMLComments(t *testing.T) {
	// 1x1 PNG.
	pngDataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO0ZQxkAAAAASUVORK5CYII="
	in := "<!DOCTYPE html><html><!-- hello --><body><img src=\"" + pngDataURL + "\"></body></html>"

	outDir := t.TempDir()
	outHTMLPath := filepath.Join(outDir, "out.html")

	newHTML, filesWritten, _, err := extractDataAssetsFromHTML(in, outHTMLPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filesWritten != 1 {
		t.Fatalf("expected 1 file written, got %d", filesWritten)
	}

	if !strings.Contains(newHTML, "<!-- hello -->") {
		t.Fatalf("expected comment preserved, got:\n%s", newHTML)
	}

	i := strings.Index(newHTML, "assets/dataasset-")
	if i < 0 {
		t.Fatalf("expected rewritten asset href/src, got:\n%s", newHTML)
	}
	j := strings.Index(newHTML[i:], ".png")
	if j < 0 {
		t.Fatalf("expected rewritten asset to be .png, got:\n%s", newHTML)
	}
	rel := newHTML[i : i+j+len(".png")]
	assetPath := filepath.Join(outDir, filepath.FromSlash(rel))

	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("expected asset file to exist at %s: %v", assetPath, err)
	}
}

func TestExtractDataAssetsFromHTML_ExtractsFontsToAssetsFolder(t *testing.T) {
	// Dummy font bytes (not a real WOFF2), but valid base64.
	fontDataURL := "data:font/woff2;base64,AA=="
	in := "<!DOCTYPE html><html><head><!-- hello --><link rel=\"preload\" as=\"font\" href=\"" + fontDataURL + "\"></head></html>"

	outDir := t.TempDir()
	outHTMLPath := filepath.Join(outDir, "out.html")

	newHTML, filesWritten, _, err := extractDataAssetsFromHTML(in, outHTMLPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filesWritten != 1 {
		t.Fatalf("expected 1 file written, got %d", filesWritten)
	}

	if !strings.Contains(newHTML, "<!-- hello -->") {
		t.Fatalf("expected comment preserved, got:\n%s", newHTML)
	}

	i := strings.Index(newHTML, "assets/dataasset-")
	if i < 0 {
		t.Fatalf("expected rewritten asset href, got:\n%s", newHTML)
	}
	j := strings.Index(newHTML[i:], ".woff2")
	if j < 0 {
		t.Fatalf("expected rewritten asset to be .woff2, got:\n%s", newHTML)
	}
	rel := newHTML[i : i+j+len(".woff2")]
	assetPath := filepath.Join(outDir, filepath.FromSlash(rel))

	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("expected asset file to exist at %s: %v", assetPath, err)
	}
}
