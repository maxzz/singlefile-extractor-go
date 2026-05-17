package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"unicode"
)

type parsedDataURL struct {
	MediaType string
	Data      []byte
	Ext       string
}

func extractDataAssetsFromHTML(htmlText string, outHTMLPath string) (newHTML string, filesWritten int, attrsReplaced int, err error) {
	outDir := filepath.Dir(outHTMLPath)
	toks := tokenizeHTML(htmlText)

	// sha256(hex prefix) -> filename
	hashToName := map[string]string{}
	writtenNames := map[string]bool{}

	var b strings.Builder
	b.Grow(len(htmlText))

	for _, tok := range toks {
		if tok.kind != "tag" {
			b.WriteString(tok.value)
			continue
		}

		tagText := tok.value
		name := tagName(tagText)
		if name == "" || isClosingTag(tagText) {
			b.WriteString(tagText)
			continue
		}

		var (
			attrName string
			rawVal   string
		)

		switch name {
		case "link":
			attrName = "href"
			rawVal = strings.TrimSpace(parseTagAttributes(tagText)["href"])
		case "img":
			attrName = "src"
			rawVal = strings.TrimSpace(parseTagAttributes(tagText)["src"])
		default:
			b.WriteString(tagText)
			continue
		}

		if rawVal == "" || !strings.HasPrefix(strings.ToLower(rawVal), "data:") {
			b.WriteString(tagText)
			continue
		}

		parsed, perr := parseDataURL(rawVal)
		if perr != nil {
			b.WriteString(tagText)
			continue
		}
		if !strings.HasPrefix(strings.ToLower(parsed.MediaType), "image/") {
			b.WriteString(tagText)
			continue
		}

		hash := shortSHA256Hex(parsed.Data, 16)
		fileName := hashToName[hash]
		if fileName == "" {
			fileName = fmt.Sprintf("dataasset-%s%s", hash, parsed.Ext)
			hashToName[hash] = fileName
		}

		if !writtenNames[fileName] {
			filePath := filepath.Join(outDir, fileName)
			if !fileExists(filePath) {
				if werr := writeFileBytes(filePath, parsed.Data); werr != nil {
					return "", 0, 0, werr
				}
				filesWritten++
			}
			writtenNames[fileName] = true
		}

		updatedTag, ok := replaceTagAttrValue(tagText, attrName, fileName)
		if ok {
			attrsReplaced++
			b.WriteString(updatedTag)
		} else {
			b.WriteString(tagText)
		}
	}

	return b.String(), filesWritten, attrsReplaced, nil
}

func parseDataURL(s string) (parsedDataURL, error) {
	raw := strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return parsedDataURL{}, fmt.Errorf("not a data url")
	}

	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return parsedDataURL{}, fmt.Errorf("invalid data url: missing comma")
	}

	header := raw[len("data:"):comma]
	payload := raw[comma+1:]

	mediaType := ""
	isBase64 := false

	parts := strings.Split(header, ";")
	for _, p := range parts {
		part := strings.TrimSpace(p)
		if part == "" {
			continue
		}
		if strings.EqualFold(part, "base64") {
			isBase64 = true
			continue
		}
		if mediaType == "" && strings.Contains(part, "/") {
			mediaType = part
		}
	}
	if mediaType == "" {
		mediaType = "text/plain"
	}

	ext := extForMediaType(mediaType)
	if ext == "" {
		ext = ".bin"
	}

	var data []byte
	if isBase64 {
		clean := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, payload)
		var decErr error
		for _, enc := range []*base64.Encoding{
			base64.StdEncoding,
			base64.RawStdEncoding,
			base64.URLEncoding,
			base64.RawURLEncoding,
		} {
			b, err := enc.DecodeString(clean)
			if err == nil {
				data = b
				decErr = nil
				break
			}
			decErr = err
		}
		if decErr != nil {
			return parsedDataURL{}, decErr
		}
	} else {
		decoded, err := url.PathUnescape(payload)
		if err != nil {
			decoded = payload
		}
		data = []byte(decoded)
	}

	return parsedDataURL{
		MediaType: mediaType,
		Data:      data,
		Ext:       ext,
	}, nil
}

func extForMediaType(mediaType string) string {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	switch mt {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		if strings.HasPrefix(mt, "image/") {
			return ".img"
		}
		return ".bin"
	}
}

func shortSHA256Hex(b []byte, nHex int) string {
	sum := sha256.Sum256(b)
	hex := fmt.Sprintf("%x", sum[:])
	if nHex <= 0 || nHex >= len(hex) {
		return hex
	}
	return hex[:nHex]
}

