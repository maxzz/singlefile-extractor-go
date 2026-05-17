# singlefile-extractor utilities (Go)

Small Go CLI utilities for extracting and post-processing content from **SingleFile-saved HTML** (often nested via `iframe[srcdoc]`).

This project is a Go port of the original Python scripts. The primary entrypoint is a single CLI binary with subcommands.

## Table of contents
- [`Install (clean machine)`](#install-clean-machine)
- [`extract`](#extract)
- [`moveout-css`](#moveout-css)
- [`format-html`](#format-html)
- [`format-css`](#format-css)
- [`extract-data-urls`](#extract-data-urls)

## `extract`

### What it does
Extracts one `<form>` element (by id) from a **SingleFile-saved HTML** and writes it into a **standalone HTML file** that preserves the form’s **visual styling**.

Specifically it:
- Walks through nested `iframe[srcdoc]` documents (SingleFile embeds pages this way).
- Finds candidate embedded documents that contain `<form id="...">`.
- Extracts from the chosen document:
  - the opening `<body ...>` tag (to keep theme/body classes)
  - all inline `<style>...</style>` blocks
  - any `<link rel="stylesheet" ...>` tags
  - the full `<form ...>...</form>` for the requested id
- Writes a new HTML file containing only those pieces.

### How to run (Windows / PowerShell)
From this repo folder:

```powershell
go run ./cmd/singlefile-extractor extract --input "Some SingleFile.html"
```

You can also run via npm:

```powershell
npm run extract -- --input "Some SingleFile.html"
npm run extract:help
```

### Options
- `-i, --input`: Path to the SingleFile-saved HTML file. (required)
- `-o, --output`: Where to write the extracted standalone HTML (default: next to `--input` with suffix `_extracted` and the same extension).
- `--form-id`: The id of the `<form>` element to extract (default: `aspnetForm`).
- `--contains`: Optional substring filter to disambiguate when multiple matches exist (example: `ESigCaptureVP.aspx`).
- `--max-depth`: Max depth to recurse through nested `iframe[srcdoc]` (default: `10`).

### Examples

```powershell
go run ./cmd/singlefile-extractor extract --input "Another SingleFile Page.html" --output "out.html"
go run ./cmd/singlefile-extractor extract --input "Some Page.html" --output "some-form.html" --form-id "myFormId"
go run ./cmd/singlefile-extractor extract --input "Some Page.html" --output "out.html" --form-id "aspnetForm" --contains "ESigCaptureVP.aspx"
```

Batch example (run on all `.html` files in a folder):

```powershell
Get-ChildItem -Filter *.html | ForEach-Object {
  $out = Join-Path $_.DirectoryName ($_.BaseName + "-extracted.html")
  go run ./cmd/singlefile-extractor extract --input $_.FullName --output $out --form-id "aspnetForm"
}
```

### Notes / limitations
- It does **not** guarantee the extracted form is fully functional (some pages rely on external scripts/services).
- It does **not** download external resources; it only keeps what is already embedded in the SingleFile HTML.

## `moveout-css`

### What it does
Moves all inline `<style>...</style>` blocks from an HTML file into a separate `.css` file, removes the `<style>` blocks from the HTML, and inserts a `<link rel="stylesheet" href="...">` back into the HTML `<head>`.

### How to run (Windows / PowerShell)
If `--output` is omitted, it writes next to the input file with suffix `.external-css` (same extension), and writes CSS to `<output>.css`.

Safe (write to new files, to a different folder):

```powershell
go run ./cmd/singlefile-extractor moveout-css --input "tests\esignature-form.html" --output "tests-local\esignature-form.external-css.html" --css-output "tests-local\esignature-form.external-css.css"
```

In-place (overwrite `--input`):

```powershell
go run ./cmd/singlefile-extractor moveout-css --input "tests\esignature-form.html" --output "tests\esignature-form.html" --css-output "tests\esignature-form.css"
```

Full CLI help:

```powershell
npm run moveout-css:help
```

## `format-html`

### What it does
Best-effort HTML formatter (pretty-printer). It tokenizes the HTML and writes it back with newlines + indentation.

By default it also runs a **CSS pipeline**:
- extracts inline `<style>...</style>` blocks into a separate CSS file (and inserts a `<link rel="stylesheet" ...>` into the formatted HTML)
- runs `extract-data-urls` on that CSS so `url(data:...)` values are moved into a vars file and referenced via `var(--...)`
- beautifies the rewritten CSS for readability

It also (by default) extracts embedded **data:** images:
- `<link href="data:image/...">` → writes an image file next to the output HTML and rewrites `href` to point at that file
- `<img src="data:image/...">` → same behavior for `src`

Disable with `--no-extract-data-assets`.

### How to run (Windows / PowerShell)
If `--output` is omitted, it writes `<input_stem>_formatted<ext>` next to the input file.

```powershell
go run ./cmd/singlefile-extractor format-html --input "tests\esignature-form.html"
```

Tip: if you omit the command, it defaults to `format-html` when `--input`/`-i` is provided:

```powershell
go run ./cmd/singlefile-extractor --input "tests\esignature-form.html"
```

Example with explicit output + indent:

```powershell
go run ./cmd/singlefile-extractor format-html --input "tests\esignature-form.html" --output "tests-local\out_formatted.html" --indent 2
```

Full CLI help:

```powershell
npm run beautify:html:help
```

### Notes / limitations
- This formatter is **not a lossless HTML parser**; it may normalize whitespace in text nodes.
- It’s intended for making “tag soup” HTML easier to read, not for producing strictly-valid HTML.

## `format-css`

### What it does
Best-effort CSS formatter (pretty-printer). It inserts newlines + indentation around `{`, `}`, and declaration `;` while respecting strings/comments and avoiding breaking tokens inside parentheses (like `url(...)`).

By default it also runs **Data URL extraction** (same logic as `extract-data-urls`):
- finds `url(data:...)` values
- moves them into a separate `:root { --... }` vars file
- rewrites the formatted CSS to reference them via `var(--...)` and adds an `@import` for the vars file

### How to run (Windows / PowerShell)
If `--output` is omitted, it writes `<input_stem>_formatted<ext>` next to the input file.

```powershell
go run ./cmd/singlefile-extractor format-css --input "tests-local\esig.smoke.css"
```

Example with explicit output + indent:

```powershell
go run ./cmd/singlefile-extractor format-css --input "tests-local\esig.smoke.css" --output "tests-local\esig.smoke_formatted.css" --indent 2
```

Full CLI help:

```powershell
npm run beautify:css:help
```

### Notes / limitations
- This formatter is **not a full CSS parser**; it may normalize whitespace and is intended for readability.

## `extract-data-urls`

### What it does
Scans a CSS file for `url(data:...)` usages, **moves those data URLs into a separate CSS file as custom properties**, and rewrites the main CSS to reference them via `var(--...)`.

It can also move existing `:root` custom properties (like `--sf-img-*`) into the vars file when their `data:` URL exceeds a configurable length threshold.

### How to run (Windows / PowerShell)
If `--output` is omitted, it writes next to the input file with suffix `_dataurls_extracted` (same extension), and writes `<output_stem>_vars<ext>` next to it.

```powershell
go run ./cmd/singlefile-extractor extract-data-urls --input "tests-local\esig.smoke_formatted.css" --output "tests-local\esig.smoke_no-dataurls.css" --vars-output "tests-local\esig.smoke_dataurls-vars.css"
```

By default, it also inserts an `@import` at the top of the rewritten CSS so the vars file is loaded automatically.

You can also run via npm:

```powershell
npm run extract:data-urls -- --input "in.css"
npm run extract:data-urls:help
```

### Notes / limitations
- Best-effort parsing (like the other formatters). Works well for typical “minified + embedded assets” CSS, but it’s not a full CSS AST parser.

## Install (clean machine)

### Prerequisites
- **Go 1.22+** (required)
- **Node.js + npm** (optional) — only needed if you want to use the `npm run ...` convenience commands

### Install Go
- **Windows (PowerShell)**: `winget install -e --id GoLang.Go`
- **macOS (Homebrew)**: `brew install go`
- **Linux**: use your distro package manager, or install from [go.dev/dl](https://go.dev/dl)

### Install Node.js (optional)
- **Windows (PowerShell)**: `winget install -e --id OpenJS.NodeJS.LTS`
- **macOS (Homebrew)**: `brew install node`
- **Linux**: install Node.js via your distro package manager, or from [nodejs.org](https://nodejs.org)

### Get the code
Clone your repo (or download a zip), then `cd` into it:

```powershell
git clone <YOUR_REPO_URL>
cd singlefile-extractor-go
```

### Build a binary
Build into `dist/`:

```powershell
npm run build
# Windows users who prefer an .exe extension:
npm run build:win
```

Or build directly with Go:

```powershell
go build -trimpath -o dist/singlefile-extractor ./cmd/singlefile-extractor
```

### Run
Without building (compiles + runs each time):

```powershell
go run ./cmd/singlefile-extractor --help
```

Run the built binary:

```powershell
# Windows
.\dist\singlefile-extractor --help
# or, if you used "npm run build:win":
.\dist\singlefile-extractor.exe --help

# macOS / Linux
./dist/singlefile-extractor --help
```

### Install into PATH (optional)

```powershell
go install ./cmd/singlefile-extractor
singlefile-extractor --help
```

On Windows you may need to add `%USERPROFILE%\go\bin` to your `PATH`.

