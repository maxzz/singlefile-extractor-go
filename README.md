# singlefile-extractor utilities (Go)

Small Go CLI utilities for extracting and post-processing content from **SingleFile-saved HTML** (often nested via `iframe[srcdoc]`).

This project is a Go port of the original Python scripts. The primary entrypoint is a single CLI binary with subcommands.

## Table of contents

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

- Command [`format-html`](#format-html)
- Command [`moveout-css`](#moveout-css)
- Command [`format-css`](#format-css)
- Command [`extract-data-urls`](#extract-data-urls)
- Command [`extract`](#extract)
- [`Install (clean machine)`](#install-clean-machine)


<h2 id="format-html">Command <code>format-html</code></h2>

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

Best-effort HTML formatter (pretty-printer). It tokenizes the HTML and writes it back with newlines + indentation.

By default it also runs a **CSS pipeline**:
- extracts inline `<style>...</style>` blocks into a separate CSS file (and inserts a `<link rel="stylesheet" ...>` into the formatted HTML)
- runs `extract-data-urls` on that CSS so `url(data:...)` values are moved into a vars file and referenced via `var(--...)`
- beautifies the rewritten CSS for readability

It also (by default) extracts embedded **data:** assets (images + fonts):
- `<link href="data:image/...">` → writes an image file to `assets/` next to the output HTML and rewrites `href` to `assets/<file>`
- `<link href="data:font/...">` → writes a font file to `assets/` next to the output HTML and rewrites `href` to `assets/<file>`
- `<img src="data:image/...">` → same behavior for `src`

Disable with `--no-extract-data-assets`.

### Options

| Option | Description |
| --- | --- |
| `-i`, `--input <path>` | Path to the HTML file to format. (required) |
| `-o`, `--output <path>` | Where to write the formatted HTML (default: next to `--input` with suffix `_formatted`). |
| `--indent <n>` | Spaces per indent level (default: `2`). |
| `--no-css-pipeline` | Disable the default CSS pipeline (format HTML only). |
| `--no-extract-data-assets` | Do not extract `data:image/...` / `data:font/...` from `<link href>` / `<img src>` into `assets/` next to the output HTML. |
| `--css-output <path>` | Where to write extracted CSS when `<style>` blocks exist (default: `<output_stem>.css`). |
| `--css-href <href>` | Override the `href` used in the inserted `<link rel=stylesheet>` tag (default: relative path to `--css-output`). |
| `--data-urls-vars-output <path>` | Where to write extracted data-url custom properties (default: `<css_stem>_dataurls-vars.css`). |
| `--data-urls-min-var-url-length <n>` | Only move existing `:root` custom properties into the vars file when the `data:` URL length is >= this value (default: `500`). |
| `--data-urls-var-prefix <prefix>` | Prefix used for generated custom properties (default: `data-url`). |
| `--data-urls-no-import` | Do not insert an `@import` for the vars file into the rewritten CSS. |
| `--data-urls-import-href <href>` | Override the `href` used in the inserted `@import` (default: relative path to vars file). |
| `-h`, `--help` | Show help. |


### Notes / limitations
- This formatter is **not a lossless HTML parser**; it may normalize whitespace in text nodes.
- It’s intended for making “tag soup” HTML easier to read, not for producing strictly-valid HTML.
- If `--output` is omitted, it writes `<input_stem>_formatted<ext>` next to the input file.

### How to run (Windows / PowerShell)

```powershell
go run ./cmd/singlefile-extractor format-html --input "tests\esignature-form.html"

# Tip: if you omit the command, it defaults to `format-html` when `--input`/`-i` is provided:
go run ./cmd/singlefile-extractor --input "tests\esignature-form.html"

# Example with explicit output + indent:
go run ./cmd/singlefile-extractor format-html --input "tests\esignature-form.html" --output "tests-local\out_formatted.html" --indent 2
```

<h2 id="moveout-css">Command <code>moveout-css</code></h2>

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

Moves all inline `<style>...</style>` blocks from an HTML file into a separate `.css` file, removes the `<style>` blocks from the HTML, and inserts a `<link rel="stylesheet" href="...">` back into the HTML `<head>`.

If `--output` is omitted, it writes next to the input file with suffix `.external-css` (same extension), and writes CSS to `<output>.css`.

### Options

| Option | Description |
| --- | --- |
| `-i`, `--input <path>` | Path to the HTML file to process. (required) |
| `-o`, `--output <path>` | Where to write the updated HTML (default: next to `--input` with suffix `.external-css`). |
| `--css-output <path>` | Where to write extracted CSS (default: `<output>.css`). |
| `--href <href>` | Optional `href` to use in the inserted `<link rel=stylesheet>` tag (default: relative path to `--css-output`). |
| `-h`, `--help` | Show help. |

### How to run (Windows / PowerShell)

```powershell
# Safe (write to new files, to a different folder):
go run ./cmd/singlefile-extractor moveout-css --input "tests\esignature-form.html" --output "tests-local\esignature-form.external-css.html" --css-output "tests-local\esignature-form.external-css.css"

# In-place (overwrite `--input`):
go run ./cmd/singlefile-extractor moveout-css --input "tests\esignature-form.html" --output "tests\esignature-form.html" --css-output "tests\esignature-form.css"
```

<h2 id="format-css">Command <code>format-css</code></h2>

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

Best-effort CSS formatter (pretty-printer). It inserts newlines + indentation around `{`, `}`, and declaration `;` while respecting strings/comments and avoiding breaking tokens inside parentheses (like `url(...)`).

By default it also runs **Data URL extraction** (same logic as `extract-data-urls`):
- finds `url(data:...)` values
- moves them into a separate `:root { --... }` vars file
- rewrites the formatted CSS to reference them via `var(--...)` and adds an `@import` for the vars file
- for `data:image/...` and `data:font/...`, writes real files into `assets/` next to the CSS and rewrites the vars file to use `url("assets/...")`

### Options

| Option | Description |
| --- | --- |
| `-i`, `--input <path>` | Path to the CSS file to format. (required) |
| `-o`, `--output <path>` | Where to write the formatted CSS (default: next to `--input` with suffix `_formatted`). |
| `--indent <n>` | Spaces per indent level (default: `2`). |
| `--no-extract-data-urls` | Disable automatic extraction of `url(data:...)` into a separate vars file. |
| `--data-urls-vars-output <path>` | Where to write extracted data-url custom properties (default: `<output_stem>_dataurls-vars.css`). |
| `--data-urls-min-var-url-length <n>` | Only move existing `:root` custom properties into vars file when the `data:` URL length is >= this value (default: `500`). |
| `--data-urls-var-prefix <prefix>` | Prefix used for generated custom properties (default: `data-url`). |
| `--data-urls-no-import` | Do not insert an `@import` for the vars file into the rewritten CSS. |
| `--data-urls-import-href <href>` | Override the href used in the inserted `@import` (default: relative path to vars file). |
| `-h`, `--help` | Show help. |

### Notes / limitations
- This formatter is **not a full CSS parser**; it may normalize whitespace and is intended for readability.
- If `--output` is omitted, it writes `<input_stem>_formatted<ext>` next to the input file.

### How to run (Windows / PowerShell)

```powershell
go run ./cmd/singlefile-extractor format-css --input "tests-local\esig.smoke.css"

# Example with explicit output + indent:
go run ./cmd/singlefile-extractor format-css --input "tests-local\esig.smoke.css" --output "tests-local\esig.smoke_formatted.css" --indent 2
```

<h2 id="extract-data-urls">Command <code>extract-data-urls</code></h2>

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

Scans a CSS file for `url(data:...)` usages, **moves those data URLs into a separate CSS file as custom properties**, and rewrites the main CSS to reference them via `var(--...)`.

It can also move existing `:root` custom properties (like `--sf-img-*`) into the vars file when their `data:` URL exceeds a configurable length threshold.

For `data:image/...` and `data:font/...` URLs it also writes real files into `assets/` (next to the vars file) and rewrites the vars to use `url("assets/...")`.

By default, it also inserts an `@import` at the top of the rewritten CSS so the vars file is loaded automatically.

### Options

| Option | Description |
| --- | --- |
| `-i`, `--input <path>` | Path to the CSS file to process. (required) |
| `-o`, `--output <path>` | Where to write the rewritten CSS (default: next to `--input` with suffix `_dataurls_extracted`). |
| `--vars-output <path>` | Where to write extracted CSS custom properties (default: next to `--output` with suffix `_vars`). |
| `--min-var-url-length <n>` | Only move existing `:root` custom properties into vars file if their `data:` URL length is >= this value (default: `500`). |
| `--var-prefix <prefix>` | Prefix used for generated custom properties (default: `data-url`). |
| `--no-import` | Do not insert an `@import` for the vars file into the rewritten CSS. |
| `--import-href <href>` | Override the href used in the inserted `@import` (default: relative path to `--vars-output`). |
| `-h`, `--help` | Show help. |

### Notes / limitations
- Best-effort parsing (like the other formatters). Works well for typical “minified + embedded assets” CSS, but it’s not a full CSS AST parser.
- If `--output` is omitted, it writes next to the input file with suffix `_dataurls_extracted` (same extension), and writes `<output_stem>_vars<ext>` next to it.

### How to run (Windows / PowerShell)

```powershell
go run ./cmd/singlefile-extractor extract-data-urls --input "tests-local\esig.smoke_formatted.css" --output "tests-local\esig.smoke_no-dataurls.css" --vars-output "tests-local\esig.smoke_dataurls-vars.css"

# You can also run via npm:
npm run extract:data-urls -- --input "in.css"
npm run extract:data-urls:help
```

<h2 id="extract">Command <code>extract</code></h2>

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

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

### Options

| Option | Description |
| --- | --- |
| `-i`, `--input <path>` | Path to the SingleFile-saved HTML file. (required) |
| `-o`, `--output <path>` | Where to write the extracted standalone HTML (default: next to `--input` with suffix `_extracted` and the same extension). |
| `--form-id <id>` | The id of the `<form>` element to extract (default: `aspnetForm`). |
| `--contains <substring>` | Optional substring filter to disambiguate when multiple matches exist (example: `ESigCaptureVP.aspx`). |
| `--max-depth <n>` | Max depth to recurse through nested `iframe[srcdoc]` (default: 10). |
| `-h`, `--help` | Show help. |

### Notes / limitations
- It does **not** guarantee the extracted form is fully functional (some pages rely on external scripts/services).
- It does **not** download external resources; it only keeps what is already embedded in the SingleFile HTML.

### How to run (Windows / PowerShell)

```powershell
# From this repo folder:
go run ./cmd/singlefile-extractor extract --input "Some SingleFile.html"

# You can also run via npm:
npm run extract -- --input "Some SingleFile.html"
npm run extract:help
```

### Examples

```powershell
go run ./cmd/singlefile-extractor extract --input "Another SingleFile Page.html" --output "out.html"
go run ./cmd/singlefile-extractor extract --input "Some Page.html" --output "some-form.html" --form-id "myFormId"
go run ./cmd/singlefile-extractor extract --input "Some Page.html" --output "out.html" --form-id "aspnetForm" --contains "ESigCaptureVP.aspx"

# Batch example (run on all `.html` files in a folder):
Get-ChildItem -Filter *.html | ForEach-Object {
  $out = Join-Path $_.DirectoryName ($_.BaseName + "-extracted.html")
  go run ./cmd/singlefile-extractor extract --input $_.FullName --output $out --form-id "aspnetForm"
}
```

<h2 id="install-clean-machine">Install (clean machine)</h2>

<!-- <hr style="border:none;border-top:2px solid #1a1a1a;margin:0 0 0.85em;" /> -->

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
```powershell
# Clone your repo (or download a zip), then `cd` into it:
git clone <YOUR_REPO_URL>
cd singlefile-extractor-go
```

### Build a binary

```powershell
# Build into `dist/`:
npm run build
# Windows users who prefer an .exe extension:
npm run build:win


# Or build directly with Go:
go build -trimpath -o dist/singlefile-extractor ./cmd/singlefile-extractor
```

### Run
```powershell
# Without building (compiles + runs each time):
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
