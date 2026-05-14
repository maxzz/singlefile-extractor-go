from __future__ import annotations

import argparse
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class DataUrlHit:
    key: str  # canonical-ish inner url contents (no surrounding quotes)
    url_token: str  # original url(...) token text to store as the variable value


def _strip_matching_quotes(s: str) -> str:
    s2 = s.strip()
    if len(s2) >= 2 and s2[0] == s2[-1] and s2[0] in {"'", '"'}:
        return s2[1:-1]
    return s2


def _is_data_protocol(inner: str) -> bool:
    return inner.lstrip().casefold().startswith("data:")


def _extract_data_url_key_and_token(url_token: str) -> DataUrlHit | None:
    """
    url_token is the full substring like: url("data:...") or url(data:...).
    Returns a key (inner value without surrounding quotes) if it is a data: URL.
    """
    m = re.match(r"(?is)\s*url\s*\((.*)\)\s*\Z", url_token)
    if not m:
        return None
    inner = m.group(1).strip()
    inner_unquoted = _strip_matching_quotes(inner)
    if not _is_data_protocol(inner_unquoted):
        return None
    return DataUrlHit(key=inner_unquoted, url_token=url_token.strip())


def _sanitize_segment(s: str) -> str:
    s = s.casefold()
    s = re.sub(r"[^a-z0-9]+", "-", s)
    s = re.sub(r"-{2,}", "-", s).strip("-")
    return s


def _selector_hint(prelude: str) -> str:
    s = prelude.strip()
    if not s:
        return "global"
    if s.startswith("@"):
        at = s[1:].split(None, 1)[0]
        return _sanitize_segment(at) or "at"

    # Use first selector from a selector list.
    first = s.split(",", 1)[0].strip()
    # Pull out a few context tokens: classes, ids, and pseudo-classes/elements.
    tokens = [m.group(0) for m in re.finditer(r"(#[A-Za-z0-9_-]+|\.[A-Za-z0-9_-]+|::?[A-Za-z0-9_-]+)", first)]
    cleaned = [_sanitize_segment(t.lstrip(".#:")) for t in tokens[-3:]]
    cleaned = [c for c in cleaned if c]
    if cleaned:
        return "-".join(cleaned)

    # Fallback: first element name.
    m = re.search(r"\b([A-Za-z][A-Za-z0-9_-]*)\b", first)
    return _sanitize_segment(m.group(1)) if m else "rule"


def _property_hint(prop: str) -> str:
    p = prop.casefold().strip()
    # Keep custom properties as-is (sans leading --).
    if p.startswith("--"):
        return _sanitize_segment(p[2:]) or "var"
    mapping = {
        "background": "bg",
        "background-image": "bg-image",
        "mask-image": "mask-image",
        "content": "content",
        "src": "src",
        "cursor": "cursor",
        "list-style": "list-style",
        "list-style-image": "list-style-image",
    }
    return mapping.get(p, _sanitize_segment(p) or "prop")


def _find_top_level_colon(statement: str) -> int | None:
    in_string: str | None = None
    escape = False
    in_comment = False
    paren_depth = 0

    i = 0
    n = len(statement)
    while i < n:
        c = statement[i]

        if in_comment:
            if c == "*" and i + 1 < n and statement[i + 1] == "/":
                i += 2
                in_comment = False
                continue
            i += 1
            continue

        if in_string is not None:
            if escape:
                escape = False
            elif c == "\\":
                escape = True
            elif c == in_string:
                in_string = None
            i += 1
            continue

        if c == "/" and i + 1 < n and statement[i + 1] == "*":
            in_comment = True
            i += 2
            continue
        if c in {"'", '"'}:
            in_string = c
            i += 1
            continue
        if c == "(":
            paren_depth += 1
            i += 1
            continue
        if c == ")":
            if paren_depth > 0:
                paren_depth -= 1
            i += 1
            continue

        if c == ":" and paren_depth == 0:
            return i
        i += 1

    return None


def _extract_property_name(lhs: str) -> str | None:
    m = re.search(r"(--[A-Za-z0-9_-]+|[A-Za-z_-][A-Za-z0-9_-]*)\s*\Z", lhs.strip())
    return m.group(1) if m else None


def _iter_url_tokens(value: str) -> list[tuple[int, int, str]]:
    """
    Return list of (start, end, url_token) for url(...) occurrences in value.
    Only finds url() outside strings/comments.
    """
    hits: list[tuple[int, int, str]] = []

    in_string: str | None = None
    escape = False
    in_comment = False

    i = 0
    n = len(value)
    while i < n:
        c = value[i]

        if in_comment:
            if c == "*" and i + 1 < n and value[i + 1] == "/":
                i += 2
                in_comment = False
                continue
            i += 1
            continue

        if in_string is not None:
            if escape:
                escape = False
            elif c == "\\":
                escape = True
            elif c == in_string:
                in_string = None
            i += 1
            continue

        # Not in comment/string.
        if c == "\\":
            # Skip escaped next char in unquoted contexts.
            i += 2 if i + 1 < n else 1
            continue

        if c == "/" and i + 1 < n and value[i + 1] == "*":
            in_comment = True
            i += 2
            continue

        if c in {"'", '"'}:
            in_string = c
            i += 1
            continue

        # Look for url(...) function.
        if (c == "u" or c == "U") and i + 2 < n and value[i : i + 3].casefold() == "url":
            j = i + 3
            while j < n and value[j].isspace():
                j += 1
            if j >= n or value[j] != "(":
                i += 1
                continue

            # Parse until matching ')'
            start = i
            j += 1  # after '('
            inner_in_string: str | None = None
            inner_escape = False
            inner_in_comment = False
            while j < n:
                ch = value[j]
                if inner_in_comment:
                    if ch == "*" and j + 1 < n and value[j + 1] == "/":
                        j += 2
                        inner_in_comment = False
                        continue
                    j += 1
                    continue
                if inner_in_string is not None:
                    if inner_escape:
                        inner_escape = False
                    elif ch == "\\":
                        inner_escape = True
                    elif ch == inner_in_string:
                        inner_in_string = None
                    j += 1
                    continue
                if ch == "\\":
                    j += 2 if j + 1 < n else 1
                    continue
                if ch == "/" and j + 1 < n and value[j + 1] == "*":
                    inner_in_comment = True
                    j += 2
                    continue
                if ch in {"'", '"'}:
                    inner_in_string = ch
                    j += 1
                    continue
                if ch == ")":
                    end = j + 1
                    hits.append((start, end, value[start:end]))
                    i = end
                    break
                j += 1
            else:
                # Unterminated url(; stop trying to parse further.
                break
            continue

        i += 1

    return hits


def _compute_import_href(*, out_css: Path, vars_css: Path) -> str:
    try:
        href = os.path.relpath(vars_css, start=out_css.parent)
    except ValueError:
        href = str(vars_css)
    return href.replace("\\", "/")


def _maybe_insert_import(css_text: str, *, href: str) -> str:
    # Avoid duplicating an identical import.
    if re.search(rf"(?im)^\s*@import\s+(?:url\()?[\"']?{re.escape(href)}[\"']?\)?\s*;", css_text):
        return css_text

    import_line = f'@import "{href}";\n'

    # If there's an @charset at the very top (ignoring whitespace/comments), it must remain first.
    prefix = ""
    rest = css_text
    ws_and_comments = re.compile(r"\A(?:\s+|/\*.*?\*/)+", flags=re.DOTALL)
    m = ws_and_comments.match(rest)
    if m:
        prefix = rest[: m.end()]
        rest = rest[m.end() :]

    if rest.lstrip().casefold().startswith("@charset"):
        m2 = re.search(r";", rest)
        if not m2:
            return prefix + rest + "\n" + import_line
        charset_stmt = rest[: m2.end()]
        after = rest[m2.end() :]
        if not charset_stmt.endswith("\n"):
            charset_stmt += "\n"
        return prefix + charset_stmt + import_line + after.lstrip("\n")

    return prefix + import_line + rest.lstrip("\n")


def parse_args(argv: list[str]) -> argparse.Namespace:
    repo_root = Path(__file__).resolve().parents[1]
    default_input = repo_root / "tests-local" / "esig.smoke_formatted.css"

    p = argparse.ArgumentParser(
        description="Extract url(data:...) occurrences into an external CSS variables file and rewrite the main CSS to reference them.",
    )
    p.add_argument(
        "-i",
        "--input",
        dest="input_path",
        type=Path,
        default=default_input,
        help="Path to the CSS file to process.",
    )
    p.add_argument(
        "-o",
        "--output",
        dest="output_path",
        type=Path,
        default=None,
        help='Where to write the rewritten CSS (default: "<input>_dataurls_extracted.css").',
    )
    p.add_argument(
        "--vars-output",
        dest="vars_output_path",
        type=Path,
        default=None,
        help='Where to write extracted CSS custom properties (default: "<output>_vars.css").',
    )
    p.add_argument(
        "--min-var-url-length",
        dest="min_var_url_length",
        type=int,
        default=500,
        help="Only move existing custom properties (e.g. --sf-img-0) into vars file if their data: URL length is >= this value (default: 500).",
    )
    p.add_argument(
        "--var-prefix",
        dest="var_prefix",
        default="data-url",
        help='Prefix used for generated custom properties (default: "data-url", results in names like --data-url-... ).',
    )
    p.add_argument(
        "--no-import",
        dest="no_import",
        action="store_true",
        help="Do not insert an @import for the vars file into the rewritten CSS.",
    )
    p.add_argument(
        "--import-href",
        dest="import_href",
        default=None,
        help="Override the href used in the inserted @import (default: relative path to --vars-output).",
    )
    return p.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)

    src: Path = args.input_path
    out_css: Path = args.output_path or src.with_name(f"{src.stem}_dataurls_extracted{src.suffix}")
    vars_css: Path = args.vars_output_path or out_css.with_name(f"{out_css.stem}_vars{out_css.suffix}")

    min_var_url_len: int = args.min_var_url_length
    if min_var_url_len < 0:
        raise ValueError("--min-var-url-length must be >= 0")

    var_prefix = _sanitize_segment(args.var_prefix)
    if not var_prefix:
        raise ValueError("--var-prefix must contain at least one alphanumeric character")

    css_text = src.read_text(encoding="utf-8", errors="replace")

    # Map canonical data-url key -> custom property name (with leading --).
    key_to_var: dict[str, str] = {}
    # Ordered list of (var_name, value)
    var_defs: list[tuple[str, str]] = []
    moved_custom_props = 0
    # Per-base counters for generated vars.
    gen_counts: dict[str, int] = {}

    def ensure_var_for_url(*, key: str, url_token: str, selector_ctx: str, prop_name: str) -> str:
        if key in key_to_var:
            return key_to_var[key]

        base = "-".join(
            part
            for part in [
                var_prefix,
                _sanitize_segment(selector_ctx),
                _property_hint(prop_name),
            ]
            if part
        )
        base = re.sub(r"-{2,}", "-", base).strip("-")
        if not base:
            base = var_prefix
        gen_counts[base] = gen_counts.get(base, 0) + 1
        var_name = f"--{base}-{gen_counts[base]}"

        key_to_var[key] = var_name
        var_defs.append((var_name, url_token))
        return var_name

    # Track current prelude for each { ... } level so we can build a selector context.
    prelude_stack: list[str] = []

    def current_selector_ctx() -> str:
        # Nearest non-empty rule-ish prelude.
        for pre in reversed(prelude_stack):
            s = pre.strip()
            if not s:
                continue
            # Prefer rule selectors over at-rules when naming.
            if not s.startswith("@"):
                return _selector_hint(s)
        # Fallback to top-most at-rule if any.
        for pre in reversed(prelude_stack):
            s = pre.strip()
            if not s:
                continue
            return _selector_hint(s)
        return "global"

    def in_root_rule() -> bool:
        for pre in reversed(prelude_stack):
            s = pre.strip()
            if not s or s.startswith("@"):
                continue
            # Crude check: if the selector list contains :root.
            if re.search(r"(?i)(^|[^\w-]):root([^\w-]|$)", s):
                return True
            return False
        return False

    def rewrite_value(value: str, *, selector_ctx: str, prop_name: str) -> str:
        # Replace each url(data:...) with var(--...).
        hits = _iter_url_tokens(value)
        if not hits:
            return value
        pieces: list[str] = []
        last = 0
        for start, end, token in hits:
            pieces.append(value[last:start])
            hit = _extract_data_url_key_and_token(token)
            if not hit:
                pieces.append(token)
            else:
                var_name = ensure_var_for_url(
                    key=hit.key,
                    url_token=hit.url_token,
                    selector_ctx=selector_ctx,
                    prop_name=prop_name,
                )
                pieces.append(f"var({var_name})")
            last = end
        pieces.append(value[last:])
        return "".join(pieces)

    def maybe_move_custom_prop(prop_name: str, value: str) -> bool:
        nonlocal moved_custom_props
        if not prop_name.startswith("--"):
            return False
        if not in_root_rule():
            return False
        value_trimmed = value.strip()
        for _, _, token in _iter_url_tokens(value):
            hit = _extract_data_url_key_and_token(token)
            if not hit:
                continue
            if len(hit.key) < min_var_url_len:
                continue
            # Move this whole custom property definition.
            moved_custom_props += 1
            # Only register as a reusable mapping if the value is exactly the url(...)
            # token (otherwise expanding var(--prop_name) could inject extra tokens).
            if value_trimmed == hit.url_token:
                key_to_var.setdefault(hit.key, prop_name)
            var_defs.append((prop_name, value_trimmed))
            return True
        return False

    # Streaming parse of the CSS. We only treat ';', '{', '}' as boundaries when
    # not in strings/comments and not in parentheses.
    out_parts: list[str] = []
    buf: list[str] = []

    in_string: str | None = None
    escape = False
    in_comment = False
    paren_depth = 0

    def flush_statement(stmt: str, *, include_semicolon: bool) -> None:
        # Try to split into declaration and rewrite values. If it doesn't look like
        # a declaration, still rewrite url(data:...) occurrences best-effort.
        colon_i = _find_top_level_colon(stmt)
        selector_ctx = current_selector_ctx()

        if colon_i is None:
            rewritten = rewrite_value(stmt, selector_ctx=selector_ctx, prop_name="statement")
            if rewritten.strip() or include_semicolon:
                out_parts.append(rewritten)
                if include_semicolon:
                    out_parts.append(";")
            return

        lhs = stmt[:colon_i]
        rhs = stmt[colon_i + 1 :]
        prop = _extract_property_name(lhs)
        if not prop:
            rewritten_rhs = rewrite_value(rhs, selector_ctx=selector_ctx, prop_name="value")
            out_parts.append(lhs)
            out_parts.append(":")
            out_parts.append(rewritten_rhs)
            if include_semicolon:
                out_parts.append(";")
            return

        # Move long data-url custom properties from :root into the vars file.
        if maybe_move_custom_prop(prop, rhs):
            return

        # Don't rewrite url() inside remaining custom property definitions by default.
        if prop.startswith("--"):
            out_parts.append(stmt)
            if include_semicolon:
                out_parts.append(";")
            return

        rewritten_rhs = rewrite_value(rhs, selector_ctx=selector_ctx, prop_name=prop)
        out_parts.append(lhs)
        out_parts.append(":")
        out_parts.append(rewritten_rhs)
        if include_semicolon:
            out_parts.append(";")

    i = 0
    n = len(css_text)
    while i < n:
        c = css_text[i]

        if in_comment:
            buf.append(c)
            if c == "*" and i + 1 < n and css_text[i + 1] == "/":
                buf.append("/")
                i += 2
                in_comment = False
                continue
            i += 1
            continue

        if in_string is not None:
            buf.append(c)
            if escape:
                escape = False
            elif c == "\\":
                escape = True
            elif c == in_string:
                in_string = None
            i += 1
            continue

        # Not in comment/string.
        if c == "\\":
            buf.append(c)
            if i + 1 < n:
                buf.append(css_text[i + 1])
                i += 2
            else:
                i += 1
            continue

        if c == "/" and i + 1 < n and css_text[i + 1] == "*":
            buf.append("/*")
            i += 2
            in_comment = True
            continue

        if c in {"'", '"'}:
            buf.append(c)
            in_string = c
            i += 1
            continue

        if c == "(":
            paren_depth += 1
            buf.append(c)
            i += 1
            continue

        if c == ")":
            if paren_depth > 0:
                paren_depth -= 1
            buf.append(c)
            i += 1
            continue

        if paren_depth == 0 and c in {";", "{", "}"}:
            stmt = "".join(buf)
            buf = []

            if c == ";":
                flush_statement(stmt, include_semicolon=True)
                i += 1
                continue

            if c == "{":
                # The buffer is the prelude (selector or at-rule).
                out_parts.append(stmt)
                out_parts.append("{")
                prelude_stack.append(stmt)
                i += 1
                continue

            if c == "}":
                # There may be a final declaration without a trailing ';'
                if stmt.strip():
                    flush_statement(stmt, include_semicolon=False)
                out_parts.append("}")
                if prelude_stack:
                    prelude_stack.pop()
                i += 1
                continue

        buf.append(c)
        i += 1

    # Trailing buffer
    trailing = "".join(buf)
    if trailing:
        out_parts.append(trailing)

    rewritten_css = "".join(out_parts)

    if not args.no_import and var_defs:
        href = args.import_href or _compute_import_href(out_css=out_css, vars_css=vars_css)
        rewritten_css = _maybe_insert_import(rewritten_css, href=href)

    # Write outputs
    out_css.parent.mkdir(parents=True, exist_ok=True)
    vars_css.parent.mkdir(parents=True, exist_ok=True)

    out_css.write_text(rewritten_css, encoding="utf-8", newline="\n")

    if var_defs:
        # De-dupe by var name, keeping the first definition encountered.
        seen_vars: set[str] = set()
        ordered_defs: list[tuple[str, str]] = []
        for name, value in var_defs:
            if name in seen_vars:
                continue
            seen_vars.add(name)
            ordered_defs.append((name, value))

        lines = [
            "/* Generated by extract-data-urls.py */",
            ":root {",
        ]
        for name, value in ordered_defs:
            lines.append(f"  {name}: {value};")
        lines.append("}")
        lines.append("")
        vars_css.write_text("\n".join(lines), encoding="utf-8", newline="\n")
    else:
        vars_css.write_text("/* No data: URLs found. */\n", encoding="utf-8", newline="\n")

    print(f"Wrote: {out_css}")
    print(f"Wrote: {vars_css}")
    print(f"- extracted vars: {len(set(n for n, _ in var_defs))}")
    print(f"- moved root custom props (min_len={min_var_url_len}): {moved_custom_props}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

