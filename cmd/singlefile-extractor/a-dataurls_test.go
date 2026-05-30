package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"singlefile-extractor-go/cmd/singlefile-extractor/utils"
)

func TestRewriteCSSExtractDataURLs_WritesAssetsForImagesAndFonts(t *testing.T) {
	// 1x1 PNG.
	pngDataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO0ZQxkAAAAASUVORK5CYII="
	// Dummy font bytes (not a real WOFF2), but valid base64.
	fontDataURL := "data:font/woff2;base64,AA=="

	css := strings.Join([]string{
		":root { --sf-img-9: url(\"" + pngDataURL + "\"); }",
		"div { background-image: url(\"" + pngDataURL + "\"); }",
		"@font-face { src: url(\"" + fontDataURL + "\") format(\"woff2\"); }",
	}, "\n")

	outDir := t.TempDir()
	outCSSPath := filepath.Join(outDir, "out.css")
	varsCSSPath := filepath.Join(outDir, "vars.css")

	res, err := rewriteCSSExtractDataURLs(css, outCSSPath, varsCSSPath, 0, "data-url", true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(res.varsCSS, "data:image") || strings.Contains(res.varsCSS, "data:font") {
		t.Fatalf("expected vars CSS to have no embedded data: URLs, got:\n%s", res.varsCSS)
	}

	re := regexp.MustCompile(`assets/dataasset-[0-9a-f]{16}\.(?:png|woff2)`)
	matches := re.FindAllString(res.varsCSS, -1)
	if len(matches) != 2 {
		t.Fatalf("expected 2 asset refs (png + woff2), got %d:\n%s", len(matches), res.varsCSS)
	}

	for _, rel := range matches {
		p := filepath.Join(outDir, filepath.FromSlash(rel))
		if !utils.FileExists(p) {
			t.Fatalf("expected asset file to exist: %s", p)
		}
	}
}

