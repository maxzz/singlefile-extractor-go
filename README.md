# singlefile-extractor utilities

Small, standard-library-only Python scripts for extracting and post-processing content from **SingleFile-saved HTML** (often nested via `iframe[srcdoc]`).

All scripts live under `scripts/`.

## Table of contents
- [`singlefile_extractor.py`](#singlefile_extractorpy)
- [`moveout-css.py`](#moveout-csspy)
- [`format-html.py`](#format-htmlpy)
- [`format-css.py`](#format-csspy)
- [`extract-data-urls.py`](#extract-data-urlspy)

## `singlefile_extractor.py`

### What it does
Extracts one `<form>` element (by id) from a **SingleFile-saved HTML** and writes it into a **standalone HTML file** that preserves the form’s **visual styling**.

Specifically it:
- Walks through nested `iframe[srcdoc]` documents (SingleFile embeds pages this way).
- Finds candidate embedded documents that contain `<form id="...">`.
- Extracts from the chosen document:
  - the opening `<body ...>` tag (to keep theme/body classes)
  - all inline `<style>...</style>` blocks
  - the full `<form ...>...</form>` for the requested id
- Writes a new HTML file containing only those pieces.

### How to run (Windows / PowerShell)
From this repo folder:

```powershell
python .\scripts\singlefile_extractor.py
```

By default, it reads `tests/Opcenter Execution (4_28_2026 3：06：53 PM).html` and writes `tests/esignature-form.html`.

You can also run via npm:

```powershell
npm run extract
npm run extract:help
```

### Options
- `-i, --input`: Path to the SingleFile-saved HTML file.
- `-o, --output`: Where to write the extracted standalone HTML.
- `--form-id`: The id of the `<form>` element to extract (default: `aspnetForm`).
- `--contains`: Optional substring filter to disambiguate when multiple matches exist (example: `ESigCaptureVP.aspx`).
- `--max-depth`: Max depth to recurse through nested `iframe[srcdoc]` (default: `10`).

To see the full CLI help:

```powershell
python .\scripts\singlefile_extractor.py --help
```

### Examples

```powershell
python .\scripts\singlefile_extractor.py --input "Another SingleFile Page.html" --output "out.html"
python .\scripts\singlefile_extractor.py --input "Some Page.html" --output "some-form.html" --form-id "myFormId"
python .\scripts\singlefile_extractor.py --input "Some Page.html" --output "out.html" --form-id "aspnetForm" --contains "ESigCaptureVP.aspx"
```

Batch example (run on all `.html` files in a folder):

```powershell
Get-ChildItem -Filter *.html | ForEach-Object {
  $out = Join-Path $_.DirectoryName ($_.BaseName + "-extracted.html")
  python .\scripts\singlefile_extractor.py --input $_.FullName --output $out --form-id "aspnetForm"
}
```

### Notes / limitations
- It does **not** guarantee the extracted form is fully functional (some pages rely on external scripts/services).
- It does **not** download external resources; it only keeps what is already embedded in the SingleFile HTML.

## `moveout-css.py`

### What it does
Moves all inline `<style>...</style>` blocks from an HTML file into a separate `.css` file, removes the `<style>` blocks from the HTML, and inserts a `<link rel="stylesheet" href="...">` back into the HTML `<head>`.

### How to run (Windows / PowerShell)
Safe (write to new files):

```powershell
python .\scripts\moveout-css.py --input "tests\esignature-form.html" --output "tests-local\esignature-form.external-css.html" --css-output "tests-local\esignature-form.external-css.css"
```

In-place (overwrites `--input`):

```powershell
python .\scripts\moveout-css.py --input "tests\esignature-form.html"
```

### Options
- `-i, --input`: Path to the HTML file to process.
- `-o, --output`: Where to write the updated HTML (default: overwrite `--input`).
- `--css-output`: Where to write extracted CSS (default: `<output>.css`).
- `--href`: Optional `href` to use in the inserted `<link>` (default: relative path to `--css-output`).

Full CLI help:

```powershell
python .\scripts\moveout-css.py --help
```

## `format-html.py`

### What it does
Best-effort HTML formatter (pretty-printer). It tokenizes the HTML and writes it back with newlines + indentation.

By default it also runs a **CSS pipeline**:
- extracts inline `<style>...</style>` blocks into a separate CSS file (and inserts a `<link rel="stylesheet" ...>` into the formatted HTML)
- runs `extract-data-urls.py` on that CSS so `url(data:...)` values are moved into a vars file and referenced via `var(--...)`
- the resulting CSS is linked from the formatted HTML (and the CSS imports the vars file)

### How to run (Windows / PowerShell)
If `--output` is omitted, it writes `<input_stem>_formatted.html` next to the input file.

```powershell
python .\scripts\format-html.py --input "tests\esignature-form.html"
```

Example with explicit output + indent:

```powershell
python .\scripts\format-html.py --input "tests\esignature-form.html" --output "tests-local\out_formatted.html" --indent 2
```

### Options
- `-i, --input`: Path to the HTML file to format.
- `-o, --output`: Where to write the formatted HTML (default: `<input>_formatted.html`).
- `--indent`: Spaces per indent level (default: `2`).
- `--no-css-pipeline`: Disable the CSS pipeline (format HTML only).
- `--css-output`: Where to write extracted CSS when `<style>` blocks exist (default: `<output_stem>.css`).
- `--css-href`: Override the href used in the inserted `<link>` tag.
- `--data-urls-min-var-url-length`: Threshold for moving existing `:root` vars (default: `500`).

Full CLI help:

```powershell
python .\scripts\format-html.py --help
```

### Notes / limitations
- This formatter is **not a lossless HTML parser**; it may normalize whitespace in text nodes.
- It’s intended for making “tag soup” HTML easier to read, not for producing strictly-valid HTML.

## `format-css.py`

### What it does
Best-effort CSS formatter (pretty-printer). It inserts newlines + indentation around `{`, `}`, and declaration `;` while respecting strings/comments and avoiding breaking tokens inside parentheses (like `url(...)`).

By default it also runs **Data URL extraction** (same logic as `extract-data-urls.py`):
- finds `url(data:...)` values
- moves them into a separate `:root { --... }` vars file
- rewrites the formatted CSS to reference them via `var(--...)` and adds an `@import` for the vars file

### How to run (Windows / PowerShell)
If `--output` is omitted, it writes `<input_stem>_formatted.css` next to the input file.

```powershell
python .\scripts\format-css.py --input "tests-local\esig.smoke.css"
```

Example with explicit output + indent:

```powershell
python .\scripts\format-css.py --input "tests-local\esig.smoke.css" --output "tests-local\esig.smoke_formatted.css" --indent 2
```

### Options
- `-i, --input`: Path to the CSS file to format.
- `-o, --output`: Where to write the formatted CSS (default: `<input>_formatted.css`).
- `--indent`: Spaces per indent level (default: `2`).
- `--no-extract-data-urls`: Disable Data URL extraction (formatting only).
- `--data-urls-vars-output`: Where to write extracted vars CSS (default: `<output_stem>_dataurls-vars.css`).
- `--data-urls-min-var-url-length`: Threshold for moving existing `:root` vars (default: `500`).

Full CLI help:

```powershell
python .\scripts\format-css.py --help
```

### Notes / limitations
- This formatter is **not a full CSS parser**; it may normalize whitespace and is intended for readability.

## `extract-data-urls.py`

### What it does
Scans a CSS file for `url(data:...)` usages, **moves those data URLs into a separate CSS file as custom properties**, and rewrites the main CSS to reference them via `var(--...)`.

It can also move existing `:root` custom properties (like `--sf-img-*`) into the vars file when their `data:` URL exceeds a configurable length threshold.

### How to run (Windows / PowerShell)

```powershell
python .\scripts\extract-data-urls.py --input "tests-local\esig.smoke_formatted.css" --output "tests-local\esig.smoke_no-dataurls.css" --vars-output "tests-local\esig.smoke_dataurls-vars.css"
```

By default, it also inserts an `@import` at the top of the rewritten CSS so the vars file is loaded automatically.

You can also run via npm:

```powershell
npm run extract:data-urls
npm run extract:data-urls:help
```

### Options
- `-i, --input`: Path to the CSS file to process.
- `-o, --output`: Where to write the rewritten CSS (default: `<input>_dataurls_extracted.css`).
- `--vars-output`: Where to write extracted CSS custom properties (default: `<output>_vars.css`).
- `--min-var-url-length`: Only move existing `:root` custom properties into the vars file if the `data:` URL length is >= this value (default: `500`).
- `--var-prefix`: Prefix used for generated custom properties (default: `data-url` → names like `--data-url-...`).
- `--no-import`: Do not insert an `@import` into the rewritten CSS.
- `--import-href`: Override the href used in the inserted `@import`.

Full CLI help:

```powershell
python .\scripts\extract-data-urls.py --help
```

### Notes / limitations
- Best-effort parsing (like the other formatters). Works well for typical “minified + embedded assets” CSS, but it’s not a full CSS AST parser.

