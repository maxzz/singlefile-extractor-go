from __future__ import annotations

import argparse
import html as html_escape
import os
import re
import sys
from pathlib import Path


STYLE_BLOCK_RE = re.compile(r"<style\b[^>]*>(.*?)</style>", flags=re.IGNORECASE | re.DOTALL)
HEAD_CLOSE_RE = re.compile(r"</head\s*>", flags=re.IGNORECASE)
HEAD_OPEN_RE = re.compile(r"<head\b[^>]*>", flags=re.IGNORECASE)
HTML_OPEN_RE = re.compile(r"<html\b[^>]*>", flags=re.IGNORECASE)
BODY_OPEN_RE = re.compile(r"<body\b[^>]*>", flags=re.IGNORECASE)


def extract_style_contents(html_text: str) -> list[str]:
    return [m.group(1) for m in STYLE_BLOCK_RE.finditer(html_text)]


def remove_style_blocks(html_text: str) -> str:
    return STYLE_BLOCK_RE.sub("", html_text)


def has_stylesheet_link(html_text: str, *, href: str) -> bool:
    escaped = re.escape(href)
    link_re = re.compile(
        rf"<link\b[^>]*\brel\s*=\s*(?:\"stylesheet\"|'stylesheet'|stylesheet)\b[^>]*\bhref\s*=\s*(?:\"{escaped}\"|'{escaped}'|{escaped})(?=[\s>/])",
        flags=re.IGNORECASE,
    )
    return link_re.search(html_text) is not None


def insert_stylesheet_link(html_text: str, *, href: str) -> str:
    if has_stylesheet_link(html_text, href=href):
        return html_text

    link_tag = f'<link rel="stylesheet" href="{html_escape.escape(href, quote=True)}">'

    m = HEAD_CLOSE_RE.search(html_text)
    if m:
        prefix = html_text[: m.start()]
        suffix = html_text[m.start() :]
        if not prefix.endswith("\n"):
            prefix += "\n"
        return prefix + link_tag + "\n" + suffix

    # No </head>. Fall back to adding inside/creating a head element.
    m = HEAD_OPEN_RE.search(html_text)
    if m:
        insert_at = m.end()
        prefix = html_text[:insert_at]
        suffix = html_text[insert_at:]
        if not prefix.endswith("\n"):
            prefix += "\n"
        return prefix + link_tag + "\n" + suffix

    head_block = f"<head>\n{link_tag}\n</head>\n"

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


def parse_args(argv: list[str]) -> argparse.Namespace:
    repo_root = Path(__file__).resolve().parents[1]
    default_input = repo_root / "tests" / "esignature-form.html"

    p = argparse.ArgumentParser(
        description="Move inline <style> CSS blocks into a separate .css file and link it from the HTML.",
    )
    p.add_argument(
        "-i",
        "--input",
        dest="input_path",
        type=Path,
        default=default_input,
        help="Path to the HTML file to process.",
    )
    p.add_argument(
        "-o",
        "--output",
        dest="output_path",
        type=Path,
        default=None,
        help="Where to write the updated HTML (default: overwrite --input).",
    )
    p.add_argument(
        "--css-output",
        dest="css_output_path",
        type=Path,
        default=None,
        help="Where to write extracted CSS (default: <output>.css).",
    )
    p.add_argument(
        "--href",
        dest="href",
        default=None,
        help="Optional href to use in the inserted <link> tag (default: relative path to --css-output).",
    )
    return p.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)

    src: Path = args.input_path
    out_html: Path = args.output_path or src
    css_out: Path = args.css_output_path or out_html.with_suffix(".css")

    # Ensure output folders exist (useful when writing to a new folder).
    out_html.parent.mkdir(parents=True, exist_ok=True)
    css_out.parent.mkdir(parents=True, exist_ok=True)

    html_text = src.read_text(encoding="utf-8", errors="replace")

    css_chunks = extract_style_contents(html_text)
    if not css_chunks:
        raise RuntimeError(f"No <style> blocks found in: {src}")

    css_text = "\n\n".join(css_chunks).rstrip() + "\n"
    css_out.write_text(css_text, encoding="utf-8", newline="\n")

    html_no_styles = remove_style_blocks(html_text)
    href = args.href or compute_default_href(out_html=out_html, css_out=css_out)
    html_out = insert_stylesheet_link(html_no_styles, href=href)
    out_html.write_text(html_out, encoding="utf-8", newline="\n")

    print(f"Wrote: {out_html}")
    print(f"Wrote: {css_out}")
    print(f"- extracted style blocks: {len(css_chunks)}")
    print(f"- css chars: {len(css_text)}")
    print(f"- link href: {href}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

