from __future__ import annotations

import argparse
import html as html_escape
import os
import re
import runpy
import sys
from dataclasses import dataclass
from pathlib import Path


VOID_ELEMENTS = {
    "area",
    "base",
    "br",
    "col",
    "embed",
    "hr",
    "img",
    "input",
    "link",
    "meta",
    "param",
    "source",
    "track",
    "wbr",
}

RAWTEXT_ELEMENTS = {"script", "style", "textarea", "pre"}

STYLE_BLOCK_RE = re.compile(
    r"^[ \t]*<style\b[^>]*>(.*?)</style>[ \t]*\r?\n?",
    flags=re.IGNORECASE | re.DOTALL | re.MULTILINE,
)
HEAD_CLOSE_RE = re.compile(r"</head\s*>", flags=re.IGNORECASE)
HEAD_OPEN_RE = re.compile(r"<head\b[^>]*>", flags=re.IGNORECASE)
HTML_OPEN_RE = re.compile(r"<html\b[^>]*>", flags=re.IGNORECASE)
BODY_OPEN_RE = re.compile(r"<body\b[^>]*>", flags=re.IGNORECASE)
LINK_STYLESHEET_TAG_RE = re.compile(
    r"<link\b[^>]*\brel\s*=\s*(?:\"stylesheet\"|'stylesheet'|stylesheet)\b[^>]*>",
    flags=re.IGNORECASE,
)

IMPLIED_CLOSE_ON_OPEN: dict[str, set[str]] = {
    # Common optional-end-tag elements.
    "li": {"li"},
    "dt": {"dt", "dd"},
    "dd": {"dt", "dd"},
    "option": {"option"},
    "optgroup": {"option"},
    # Table elements (HTML often omits </td>, </th>, </tr>, etc).
    "td": {"td", "th"},
    "th": {"td", "th"},
    "tr": {"td", "th", "tr"},
    "thead": {"td", "th", "tr", "thead", "tbody", "tfoot"},
    "tbody": {"td", "th", "tr", "thead", "tbody", "tfoot"},
    "tfoot": {"td", "th", "tr", "thead", "tbody", "tfoot"},
}


@dataclass(frozen=True)
class Token:
    kind: str  # "tag" | "text"
    value: str


def _parse_tag(html_text: str, start: int) -> tuple[str, int]:
    """Parse a tag starting at '<' and return (tag_text, next_index)."""
    n = len(html_text)
    i = start + 1
    quote: str | None = None
    while i < n:
        c = html_text[i]
        if quote:
            if c == quote:
                quote = None
        else:
            if c in {"'", '"'}:
                quote = c
            elif c == ">":
                return html_text[start : i + 1], i + 1
        i += 1
    return html_text[start:], n


def _tag_name(tag_text: str) -> str | None:
    s = tag_text.strip()
    if not s.startswith("<") or len(s) < 3:
        return None
    if s.startswith(("<!--", "<!", "<?")):
        return None
    if s.startswith("</"):
        s2 = s[2:]
    else:
        s2 = s[1:]
    s2 = s2.lstrip()
    m = re.match(r"([a-zA-Z][a-zA-Z0-9:_-]*)", s2)
    return m.group(1).casefold() if m else None


def _is_closing_tag(tag_text: str) -> bool:
    return tag_text.lstrip().startswith("</")


def _is_self_closing_tag(tag_text: str) -> bool:
    s = tag_text.strip()
    return s.endswith("/>")


def tokenize_html(html_text: str) -> list[Token]:
    tokens: list[Token] = []
    i = 0
    n = len(html_text)

    while i < n:
        if html_text[i] != "<":
            j = html_text.find("<", i)
            if j < 0:
                j = n
            tokens.append(Token("text", html_text[i:j]))
            i = j
            continue

        # Comment
        if html_text.startswith("<!--", i):
            j = html_text.find("-->", i + 4)
            if j < 0:
                tokens.append(Token("tag", html_text[i:]))
                break
            tokens.append(Token("tag", html_text[i : j + 3]))
            i = j + 3
            continue

        tag_text, next_i = _parse_tag(html_text, i)
        tokens.append(Token("tag", tag_text))
        i = next_i

        # If this is an opening rawtext element, don't try to tokenize its contents.
        name = _tag_name(tag_text)
        if not name:
            continue
        if _is_closing_tag(tag_text):
            continue
        if _is_self_closing_tag(tag_text) or name in VOID_ELEMENTS:
            continue
        if name not in RAWTEXT_ELEMENTS:
            continue

        close_re = re.compile(rf"</\s*{re.escape(name)}\s*>", flags=re.IGNORECASE)
        m = close_re.search(html_text, i)
        if not m:
            # No closing tag found; treat rest as text.
            tokens.append(Token("text", html_text[i:]))
            break
        tokens.append(Token("text", html_text[i : m.start()]))
        tokens.append(Token("tag", html_text[m.start() : m.end()]))
        i = m.end()

    return tokens


def _pop_through_nearest(stack: list[str], names: set[str]) -> None:
    """Pop stack down through the nearest open element in names (inclusive)."""
    for idx in range(len(stack) - 1, -1, -1):
        if stack[idx] in names:
            del stack[idx:]
            return


def _pop_close_tag(stack: list[str], name: str) -> int:
    """Pop stack to close name; returns indent level for printing the closing tag."""
    for idx in range(len(stack) - 1, -1, -1):
        if stack[idx] == name:
            del stack[idx:]
            return idx
    return len(stack)


def format_html(html_text: str, *, indent: str = "  ") -> str:
    tokens = tokenize_html(html_text)

    lines: list[str] = []
    stack: list[str] = []

    def emit(line: str) -> None:
        if line == "":
            return
        lines.append(line)

    for tok in tokens:
        if tok.kind == "text":
            # Skip pure whitespace between tags.
            if tok.value.strip() == "":
                continue
            for raw_line in tok.value.splitlines():
                s = raw_line.strip()
                if s == "":
                    continue
                emit(f"{indent * len(stack)}{s}")
            continue

        tag = tok.value.strip()
        if tag == "":
            continue

        name = _tag_name(tag)

        if _is_closing_tag(tag):
            if name:
                indent_level = _pop_close_tag(stack, name)
            else:
                indent_level = len(stack)
            emit(f"{indent * indent_level}{tag}")
            continue

        if not name:
            emit(f"{indent * len(stack)}{tag}")
            continue

        # Apply implied end-tag rules (improves indentation for common optional-end-tag
        # patterns, especially inside tables).
        implied = IMPLIED_CLOSE_ON_OPEN.get(name)
        if implied:
            _pop_through_nearest(stack, implied)

        emit(f"{indent * len(stack)}{tag}")

        if _is_self_closing_tag(tag) or name in VOID_ELEMENTS:
            continue
        stack.append(name)

    return "\n".join(lines).rstrip() + "\n"


def extract_style_contents(html_text: str) -> list[str]:
    return [m.group(1) for m in STYLE_BLOCK_RE.finditer(html_text)]


def remove_style_blocks(html_text: str) -> str:
    return STYLE_BLOCK_RE.sub("", html_text)


def _has_stylesheet_link(html_text: str, *, href: str) -> bool:
    escaped = re.escape(href)
    link_re = re.compile(
        rf"<link\b[^>]*\brel\s*=\s*(?:\"stylesheet\"|'stylesheet'|stylesheet)\b[^>]*\bhref\s*=\s*(?:\"{escaped}\"|'{escaped}'|{escaped})(?=[\s>/])",
        flags=re.IGNORECASE,
    )
    return link_re.search(html_text) is not None


def insert_stylesheet_link(html_text: str, *, href: str, indent_unit: str = "  ") -> str:
    if _has_stylesheet_link(html_text, href=href):
        return html_text

    link_tag = f'<link rel="stylesheet" href="{html_escape.escape(href, quote=True)}">'

    m = HEAD_CLOSE_RE.search(html_text)
    if m:
        # Insert before the </head> *line* to preserve indentation, and avoid leaving
        # an "indent-only" blank line behind.
        line_start = html_text.rfind("\n", 0, m.start())
        if line_start < 0:
            line_start = 0
        else:
            line_start += 1

        closing_indent = html_text[line_start : m.start()]
        if closing_indent.strip() != "":
            closing_indent = ""
        child_indent = closing_indent + indent_unit

        prefix = html_text[:line_start]
        suffix = html_text[line_start:]
        return prefix + child_indent + link_tag + "\n" + suffix

    # No </head>. Fall back to adding inside/creating a head element.
    m = HEAD_OPEN_RE.search(html_text)
    if m:
        insert_at = m.end()
        prefix = html_text[:insert_at]
        suffix = html_text[insert_at:]

        line_start = html_text.rfind("\n", 0, m.start())
        if line_start < 0:
            line_start = 0
        else:
            line_start += 1
        head_indent = html_text[line_start : m.start()]
        if head_indent.strip() != "":
            head_indent = ""
        child_indent = head_indent + indent_unit

        if not prefix.endswith("\n"):
            prefix += "\n"
        if suffix.startswith("\n"):
            suffix = suffix[1:]
        return prefix + child_indent + link_tag + "\n" + suffix

    head_block = f"<head>\n{indent_unit}{link_tag}\n</head>\n"

    m = BODY_OPEN_RE.search(html_text)
    if m:
        return html_text[: m.start()] + head_block + html_text[m.start() :]

    m = HTML_OPEN_RE.search(html_text)
    if m:
        insert_at = m.end()
        prefix = html_text[:insert_at]
        suffix = html_text[insert_at:]
        if not prefix.endswith("\n"):
            prefix += "\n"
        return prefix + head_block + suffix

    return head_block + html_text


def compute_default_href(*, out_html: Path, css_out: Path) -> str:
    try:
        href = os.path.relpath(css_out, start=out_html.parent)
    except ValueError:
        href = str(css_out)
    return href.replace("\\", "/")


def iter_linked_stylesheet_hrefs(html_text: str) -> list[str]:
    hrefs: list[str] = []
    for m in LINK_STYLESHEET_TAG_RE.finditer(html_text):
        tag = m.group(0)
        hm = re.search(r"\bhref\s*=\s*(?:\"([^\"]*)\"|'([^']*)'|([^\s>]+))", tag, flags=re.IGNORECASE)
        if not hm:
            continue
        href = (hm.group(1) or hm.group(2) or hm.group(3) or "").strip()
        if href:
            hrefs.append(href)
    return hrefs


def _default_output_path(input_path: Path) -> Path:
    return input_path.with_name(f"{input_path.stem}_formatted{input_path.suffix}")


def collapse_blank_lines(text: str, *, max_consecutive: int = 2) -> str:
    if max_consecutive < 0:
        return text
    if text == "":
        return text

    out: list[str] = []
    blank_run = 0
    for line in text.splitlines(keepends=True):
        if line.strip() == "":
            blank_run += 1
            if blank_run > max_consecutive:
                continue
        else:
            blank_run = 0
        out.append(line)

    result = "".join(out)
    if result != "" and not result.endswith("\n"):
        result += "\n"
    return result


def parse_args(argv: list[str]) -> argparse.Namespace:
    repo_root = Path(__file__).resolve().parents[1]
    default_input = repo_root / "tests" / "esignature-form.html"

    p = argparse.ArgumentParser(
        description="Format (pretty-print) an HTML file with indentation.",
    )
    p.add_argument(
        "-i",
        "--input",
        dest="input_path",
        type=Path,
        default=default_input,
        help="Path to the HTML file to format.",
    )
    p.add_argument(
        "-o",
        "--output",
        dest="output_path",
        type=Path,
        default=None,
        help='Where to write the formatted HTML (default: "<input>_formatted.html").',
    )
    p.add_argument(
        "--indent",
        dest="indent",
        type=int,
        default=2,
        help="Spaces per indent level (default: 2).",
    )
    p.add_argument(
        "--no-css-pipeline",
        dest="no_css_pipeline",
        action="store_true",
        help="Disable the default CSS pipeline (extract <style> blocks, run extract-data-urls, and link CSS).",
    )
    p.add_argument(
        "--css-output",
        dest="css_output_path",
        type=Path,
        default=None,
        help='Where to write extracted CSS when <style> blocks exist (default: "<output_stem>.css").',
    )
    p.add_argument(
        "--css-href",
        dest="css_href",
        default=None,
        help="Override the href used in the inserted <link rel=stylesheet> tag (default: relative path to --css-output).",
    )
    p.add_argument(
        "--data-urls-vars-output",
        dest="data_urls_vars_output",
        type=Path,
        default=None,
        help='Where to write extracted data-url custom properties (default: "<css_stem>_dataurls-vars.css").',
    )
    p.add_argument(
        "--data-urls-min-var-url-length",
        dest="data_urls_min_var_url_length",
        type=int,
        default=500,
        help="Only move existing :root custom properties into vars file when the data: URL length is >= this value (default: 500).",
    )
    p.add_argument(
        "--data-urls-var-prefix",
        dest="data_urls_var_prefix",
        default="data-url",
        help='Prefix used for generated custom properties (default: "data-url").',
    )
    p.add_argument(
        "--data-urls-no-import",
        dest="data_urls_no_import",
        action="store_true",
        help="Do not insert an @import for the vars file into the rewritten CSS.",
    )
    p.add_argument(
        "--data-urls-import-href",
        dest="data_urls_import_href",
        default=None,
        help="Override the href used in the inserted @import (default: relative path to vars file).",
    )
    return p.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)

    src: Path = args.input_path
    out: Path = args.output_path or _default_output_path(src)
    indent_spaces: int = args.indent
    if indent_spaces < 0:
        raise ValueError("--indent must be >= 0")
    indent_unit = " " * indent_spaces

    html_text = src.read_text(encoding="utf-8", errors="replace")
    formatted = format_html(html_text, indent=indent_unit)
    formatted = collapse_blank_lines(formatted, max_consecutive=2)

    if args.no_css_pipeline:
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(formatted, encoding="utf-8", newline="\n")
        print(f"Wrote: {out}")
        print(f"- input: {src}")
        print(f"- indent: {indent_spaces} spaces")
        print(f"- chars: {len(formatted)}")
        return 0

    # Default CSS pipeline:
    # 1) Extract <style> blocks -> CSS file + <link>
    # 2) Run extract-data-urls.py on that CSS (or on linked CSS if no <style> blocks)
    css_files: list[Path] = []
    vars_files: list[Path] = []

    style_chunks = extract_style_contents(formatted)
    if style_chunks:
        css_out = args.css_output_path or out.with_suffix(".css")
        css_out.parent.mkdir(parents=True, exist_ok=True)

        css_text = "\n\n".join(style_chunks).rstrip() + "\n"
        css_out.write_text(css_text, encoding="utf-8", newline="\n")

        href = args.css_href or compute_default_href(out_html=out, css_out=css_out)
        html_no_styles = remove_style_blocks(formatted)
        html_linked = insert_stylesheet_link(html_no_styles, href=href, indent_unit=indent_unit)
        html_linked = collapse_blank_lines(html_linked, max_consecutive=2)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(html_linked, encoding="utf-8", newline="\n")

        css_files.append(css_out)
    else:
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(formatted, encoding="utf-8", newline="\n")
        # No inline <style> blocks. Try to run data-url extraction on any local linked stylesheets.
        for href in iter_linked_stylesheet_hrefs(formatted):
            h = href.strip()
            if not h:
                continue
            if re.match(r"(?i)^(?:https?:)?//", h) or h.casefold().startswith("data:"):
                continue
            # Resolve relative to output HTML location.
            css_path = Path(h)
            if not css_path.is_absolute():
                css_path = (out.parent / css_path).resolve()
            if css_path.exists() and css_path.is_file():
                css_files.append(css_path)

    if css_files:
        extractor_path = Path(__file__).resolve().with_name("extract-data-urls.py")
        if not extractor_path.exists():
            raise RuntimeError(f"Could not locate extract-data-urls.py next to this script: {extractor_path}")
        extractor_globals = runpy.run_path(str(extractor_path))
        extractor_main = extractor_globals.get("main")
        if not callable(extractor_main):
            raise RuntimeError("extract-data-urls.py did not expose a callable main(argv) function.")

        min_len: int = args.data_urls_min_var_url_length
        if min_len < 0:
            raise ValueError("--data-urls-min-var-url-length must be >= 0")

        if args.data_urls_vars_output and len(css_files) > 1:
            raise ValueError("--data-urls-vars-output can only be used when processing a single CSS file.")

        for css_path in css_files:
            vars_out = args.data_urls_vars_output or css_path.with_name(f"{css_path.stem}_dataurls-vars{css_path.suffix}")
            extractor_argv = [
                "--input",
                str(css_path),
                "--output",
                str(css_path),  # rewrite in-place
                "--vars-output",
                str(vars_out),
                "--min-var-url-length",
                str(min_len),
                "--var-prefix",
                str(args.data_urls_var_prefix),
            ]
            if args.data_urls_no_import:
                extractor_argv.append("--no-import")
            if args.data_urls_import_href:
                extractor_argv.extend(["--import-href", str(args.data_urls_import_href)])

            extractor_main(extractor_argv)
            vars_files.append(vars_out)

        # Finally, beautify the rewritten CSS so the pipeline output matches `format-css.py`
        # quality (extract-data-urls.py focuses on transformations, not formatting).
        formatter_path = Path(__file__).resolve().with_name("format-css.py")
        if not formatter_path.exists():
            raise RuntimeError(f"Could not locate format-css.py next to this script: {formatter_path}")
        formatter_globals = runpy.run_path(str(formatter_path))
        format_css = formatter_globals.get("format_css")
        if not callable(format_css):
            raise RuntimeError("format-css.py did not expose a callable format_css(css_text, *, indent) function.")

        css_indent = " " * indent_spaces
        for css_path in css_files:
            css_text = css_path.read_text(encoding="utf-8", errors="replace")
            formatted_css = format_css(css_text, indent=css_indent)
            css_path.write_text(formatted_css, encoding="utf-8", newline="\n")

    # Keep this script's own summary short (extractor prints details).
    print(f"Wrote: {out}")
    print(f"- input: {src}")
    print(f"- indent: {indent_spaces} spaces")
    if css_files:
        print(f"- css files processed: {len(css_files)}")
    if vars_files:
        print(f"- vars files written: {len(vars_files)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

