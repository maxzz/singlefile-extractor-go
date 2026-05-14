from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from html.parser import HTMLParser
from pathlib import Path


@dataclass(frozen=True)
class IFrameSrcdoc:
    id: str | None
    srcdoc: str


@dataclass(frozen=True)
class SrcdocDocument:
    """A decoded HTML document coming from some iframe's srcdoc."""

    html: str
    depth: int
    path: tuple[str, ...]


class IFrameSrcdocParser(HTMLParser):
    def __init__(self) -> None:
        # convert_charrefs=True decodes entities once in attrs/data, which matches how
        # we want to interpret nested srcdoc values at each parse layer.
        super().__init__(convert_charrefs=True)
        self.iframes: list[IFrameSrcdoc] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag.lower() != "iframe":
            return

        d = {k.lower(): v for k, v in attrs}
        if "srcdoc" not in d:
            return

        self.iframes.append(IFrameSrcdoc(id=d.get("id"), srcdoc=d.get("srcdoc") or ""))


def parse_iframe_srcdocs(html_text: str) -> list[IFrameSrcdoc]:
    parser = IFrameSrcdocParser()
    parser.feed(html_text)
    return parser.iframes


class FormIdDetector(HTMLParser):
    def __init__(self, *, form_id: str) -> None:
        super().__init__(convert_charrefs=True)
        self._target = form_id.casefold()
        self.found = False

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if self.found or tag.lower() != "form":
            return
        for k, v in attrs:
            if k.lower() == "id" and (v or "").casefold() == self._target:
                self.found = True
                return


def document_contains_form_id(html_text: str, *, form_id: str) -> bool:
    detector = FormIdDetector(form_id=form_id)
    detector.feed(html_text)
    return detector.found


def iter_srcdoc_documents(outer_html: str, *, max_depth: int = 10) -> list[SrcdocDocument]:
    """Return all nested srcdoc documents found by walking iframe[srcdoc] recursively.

    We intentionally do NOT treat the outer HTML itself as a candidate document because
    embedded documents appear inside attribute values there.
    """

    docs: list[SrcdocDocument] = []
    queue: list[SrcdocDocument] = []

    top_iframes = parse_iframe_srcdocs(outer_html)
    for idx, fr in enumerate(top_iframes):
        label = fr.id or f"iframe#{idx}"
        queue.append(SrcdocDocument(html=fr.srcdoc, depth=1, path=(label,)))

    while queue:
        cur = queue.pop(0)
        docs.append(cur)

        if cur.depth >= max_depth:
            continue

        nested = parse_iframe_srcdocs(cur.html)
        for idx, fr in enumerate(nested):
            label = fr.id or f"iframe#{idx}"
            queue.append(SrcdocDocument(html=fr.srcdoc, depth=cur.depth + 1, path=cur.path + (label,)))

    return docs


STYLE_RE = re.compile(r"<style\b[^>]*>.*?</style>", flags=re.IGNORECASE | re.DOTALL)
LINK_STYLESHEET_RE = re.compile(
    r"<link\b[^>]*\brel\s*=\s*(?:\"stylesheet\"|'stylesheet'|stylesheet)\b[^>]*>",
    flags=re.IGNORECASE,
)
BODY_OPEN_RE = re.compile(r"<body\b[^>]*>", flags=re.IGNORECASE)


def extract_form_and_styles(target_html: str, *, form_id: str) -> tuple[str, list[str], list[str], str]:
    body_m = BODY_OPEN_RE.search(target_html)
    if not body_m:
        raise RuntimeError("Could not find <body> in target srcdoc HTML.")
    body_open = body_m.group(0)

    styles = STYLE_RE.findall(target_html)
    links = LINK_STYLESHEET_RE.findall(target_html)

    # Find the <form ... id=<form_id> ...> ... </form> block.
    escaped = re.escape(form_id)
    form_id_m = re.search(
        rf"<form\b[^>]*\bid\s*=\s*(?:\"{escaped}\"|'{escaped}'|{escaped})(?=[\s>/])",
        target_html,
        flags=re.IGNORECASE,
    )
    if not form_id_m:
        raise RuntimeError(f"Could not find <form id={form_id}> in target srcdoc HTML.")

    # The regex anchors on the literal "<form", so the match start is the form tag.
    start = form_id_m.start()

    end = target_html.lower().find("</form>", form_id_m.end())
    if end < 0:
        raise RuntimeError(f"Could not locate </form> end tag for {form_id}.")

    form_html = target_html[start : end + len("</form>")]
    return body_open, styles, links, form_html


def build_output_html(*, body_open: str, styles: list[str], links: list[str], form_html: str) -> str:
    head_bits = [
        '<meta charset="utf-8">',
        '<meta name="viewport" content="width=device-width, initial-scale=1.0">',
        "<title>ESignature Form</title>",
        *links,
        *styles,
    ]
    head_html = "\n".join(head_bits)

    return (
        "<!DOCTYPE html>\n"
        "<html lang=\"en\">\n"
        "<head>\n"
        f"{head_html}\n"
        "</head>\n"
        f"{body_open}\n"
        f"{form_html}\n"
        "</body>\n"
        "</html>\n"
    )


def parse_args(argv: list[str]) -> argparse.Namespace:
    repo_root = Path(__file__).resolve().parents[1]
    default_input = repo_root / "tests" / "Opcenter Execution (4_28_2026 3：06：53 PM).html"
    default_output = repo_root / "tests" / "esignature-form.html"

    p = argparse.ArgumentParser(
        description="Extract a <form id=...> and related inline styles from a SingleFile HTML (nested iframe srcdoc) into a standalone HTML.",
    )
    p.add_argument(
        "-i",
        "--input",
        dest="input_path",
        type=Path,
        default=default_input,
        help="Path to the SingleFile-saved HTML file.",
    )
    p.add_argument(
        "-o",
        "--output",
        dest="output_path",
        type=Path,
        default=default_output,
        help="Where to write the extracted standalone HTML.",
    )
    p.add_argument(
        "--form-id",
        dest="form_id",
        default="aspnetForm",
        help="The id of the <form> element to extract (default: aspnetForm).",
    )
    p.add_argument(
        "--contains",
        dest="contains",
        default=None,
        help="Optional substring to disambiguate when multiple matching forms exist (e.g. ESigCaptureVP.aspx).",
    )
    p.add_argument(
        "--max-depth",
        dest="max_depth",
        type=int,
        default=10,
        help="Max depth to recurse through nested iframe[srcdoc] (default: 10).",
    )
    return p.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)

    src: Path = args.input_path
    out: Path = args.output_path
    form_id: str = args.form_id
    contains: str | None = args.contains
    max_depth: int = args.max_depth

    outer_html = src.read_text(encoding="utf-8", errors="replace")

    candidates = iter_srcdoc_documents(outer_html, max_depth=max_depth)
    matches = [d for d in candidates if document_contains_form_id(d.html, form_id=form_id)]

    if contains:
        matches = [d for d in matches if contains in d.html]

    if not matches:
        msg = [
            f"Could not find <form id={form_id}> inside any nested iframe[srcdoc] documents.",
            f"- input: {src}",
            f"- searched docs: {len(candidates)} (max_depth={max_depth})",
        ]
        if contains:
            msg.append(f"- contains filter: {contains!r}")
        raise RuntimeError("\n".join(msg))

    if len(matches) > 1:
        # If the user supplied a disambiguation string but we still have multiple results,
        # fail loudly with candidates.
        if contains:
            lines = [
                f"Found multiple nested documents containing <form id={form_id}> ({len(matches)} matches), even after filtering by --contains.",
                "Refine --contains to something more specific.",
                "",
                "Matches (iframe path):",
            ]
            for d in matches[:20]:
                lines.append(f"- {' > '.join(d.path)} (depth={d.depth}, chars={len(d.html)})")
            if len(matches) > 20:
                lines.append(f"... and {len(matches) - 20} more")
            raise RuntimeError("\n".join(lines))

        # Otherwise auto-select the "most specific" candidate (deepest nesting, then most
        # inline styles, then smallest doc).
        def score(d: SrcdocDocument) -> tuple[int, int, int]:
            return (d.depth, d.html.lower().count("<style"), -len(d.html))

        target_doc = max(matches, key=score)
        print(
            "Note: multiple matching documents found; auto-selected the deepest match.\n"
            "      Use --contains <substring> to force a different one if needed.\n"
            f"      Selected: {' > '.join(target_doc.path)}",
            file=sys.stderr,
        )
    else:
        target_doc = matches[0]

    body_open, styles, links, form_html = extract_form_and_styles(target_doc.html, form_id=form_id)
    output_html = build_output_html(body_open=body_open, styles=styles, links=links, form_html=form_html)

    out.write_text(output_html, encoding="utf-8", newline="\n")

    print(f"Wrote: {out}")
    print(f"- extracted form id: {form_id}")
    print(f"- source iframe path: {' > '.join(target_doc.path)}")
    print(f"- styles: {len(styles)}")
    print(f"- link rel=stylesheet: {len(links)}")
    print(f"- form chars: {len(form_html)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

