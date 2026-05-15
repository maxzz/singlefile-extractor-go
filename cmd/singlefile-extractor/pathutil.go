package main

import (
	"path/filepath"
	"strings"
)

func replaceExt(path string, newExt string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	return filepath.Join(filepath.Dir(path), base+newExt)
}

func withName(path string, newName string) string {
	return filepath.Join(filepath.Dir(path), newName)
}

func defaultFormattedPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	stem := strings.TrimSuffix(filepath.Base(inputPath), ext)
	return filepath.Join(filepath.Dir(inputPath), stem+"_formatted"+ext)
}

func defaultExtractOutputPath(inputPath string) string {
	origExt := filepath.Ext(inputPath)
	stem := strings.TrimSuffix(filepath.Base(inputPath), origExt)
	outExt := origExt
	if outExt == "" {
		outExt = ".html"
	}
	return filepath.Join(filepath.Dir(inputPath), stem+"_extracted"+outExt)
}

func defaultMoveoutCSSOutputHTMLPath(inputPath string) string {
	origExt := filepath.Ext(inputPath)
	stem := strings.TrimSuffix(filepath.Base(inputPath), origExt)
	outExt := origExt
	if outExt == "" {
		outExt = ".html"
	}
	return filepath.Join(filepath.Dir(inputPath), stem+".external-css"+outExt)
}
