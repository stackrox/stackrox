#!/usr/bin/env python3
"""Render JUnit XML files into a single HTML report (streaming edition).

Same output as junit2html.py, but built for large inputs:

* Parsing uses the streaming pull API (``xml.etree.ElementTree.iterparse``) and
  clears elements as soon as they are consumed, so a whole input file is never
  held in memory as a DOM.
* Output is written straight to the output stream as content is produced, rather
  than accumulating strings and joining them.
* Because the summary and each suite header print counts *before* their test
  cases are streamed, we make two passes: pass 1 counts (global totals and
  per-suite aggregated counts), pass 2 renders.

Bootstrap is referenced from a CDN via <link>; nothing is embedded or vendored,
so the output is small but needs network to style itself when viewed.

Standard library only.

Usage:
    junit2html_streaming.py INPUT_DIR OUTPUT_HTML [--title TITLE] [--bootstrap-url URL]

On CDATA: the content of <system-out>/<system-err> (which JUnit stores as CDATA)
is preserved verbatim and HTML-escaped into <pre>, the HTML equivalent of "treat
this text literally".
"""

import argparse
import html
import sys
from collections import Counter
from pathlib import Path
import xml.etree.ElementTree as ET

BOOTSTRAP_URL = "https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css"

# Small additions on top of Bootstrap (things Bootstrap does not provide).
EXTRA_CSS = (
    ".suite{border-left:3px solid #dee2e6;padding-left:1rem}"
    "pre{max-height:24rem;white-space:pre-wrap;word-break:break-word}"
    ".summary-badges .badge{font-size:1rem;margin-right:.5rem}"
    "summary{cursor:pointer}"
)

# status -> (row class, badge class, label). Errors are treated as failures (red).
STATUS = {
    "passed": ("table-success", "text-bg-success", "PASSED"),
    "failed": ("table-danger", "text-bg-danger", "FAILED"),
    "error": ("table-danger", "text-bg-danger", "ERROR"),
    "skipped": ("table-secondary", "text-bg-secondary", "SKIPPED"),
}

TABLE_OPEN = ('<table class="table table-sm table-hover align-middle"><thead><tr>'
              "<th>Status</th><th>Test</th><th>Class</th><th>Time (s)</th>"
              "</tr></thead><tbody>")
TABLE_CLOSE = "</tbody></table>"

esc = html.escape


def _count_bg(count, color_class):
    """Bootstrap text-bg class for a count badge: neutral (light) when the count
    is zero, coloured only when it is non-zero."""
    return color_class if count else "text-bg-light"


class NotJUnit(Exception):
    def __init__(self, tag):
        super().__init__(tag)
        self.tag = tag


def _text(el):
    """Full character content of an element (joins text and CDATA), or ''."""
    if el is None:
        return ""
    return "".join(el.itertext())


def _classify(case_elem):
    if case_elem.find("failure") is not None:
        return "failed"
    if case_elem.find("error") is not None:
        return "error"
    if case_elem.find("skipped") is not None:
        return "skipped"
    return "passed"


# --------------------------------------------------------------------------- #
# Pass 1: count. Streams each file; returns per-suite aggregated counts (in
# suite-start order) and the file totals, or raises on a bad/foreign file.
# --------------------------------------------------------------------------- #
def count_file(path):
    suites = []          # aggregated Counter per suite, in start order
    stack = []           # counters of currently open suites
    file_totals = Counter()

    it = ET.iterparse(path, events=("start", "end"))
    try:
        _, root = next(it)
    except StopIteration:
        raise ET.ParseError("empty document")
    if root.tag not in ("testsuites", "testsuite"):
        raise NotJUnit(root.tag)

    def on_start(elem):
        if elem.tag == "testsuite":
            c = Counter()
            suites.append(c)
            stack.append(c)

    def on_end(elem):
        if elem.tag == "testcase":
            if stack:
                stack[-1]["total"] += 1
                stack[-1][_classify(elem)] += 1
            elem.clear()
        elif elem.tag == "testsuite":
            c = stack.pop()
            (stack[-1] if stack else file_totals).update(c)
            elem.clear()

    on_start(root)
    for event, elem in it:
        (on_start if event == "start" else on_end)(elem)
    return suites, file_totals


def pass1(files):
    valid_files, suite_counts, totals = [], [], Counter()
    for path in files:
        try:
            suites, file_totals = count_file(path)
        except NotJUnit as e:
            print(f"{path}: not a JUnit report (root <{e.tag}>); skipping",
                  file=sys.stderr)
            continue
        except ET.ParseError as e:
            print(f"{path}: not well-formed XML ({e}); skipping", file=sys.stderr)
            continue
        valid_files.append(path)
        suite_counts.extend(suites)   # global order == pass-2 suite-start order
        totals.update(file_totals)
    return valid_files, suite_counts, totals


# --------------------------------------------------------------------------- #
# Pass 2: render, streaming to the output file object.
# --------------------------------------------------------------------------- #
class Renderer:
    def __init__(self, out, suite_counts):
        self.out = out
        self.suite_counts = suite_counts
        self.sid = 0                 # next suite id (matches pass-1 start order)
        self.stack = []              # frames: {"table_open": bool}
        self.in_testcase = False

    def handle_start(self, elem):
        if elem.tag == "testsuite":
            counts = self.suite_counts[self.sid]
            self.sid += 1
            if self.stack:           # a nested suite is a non-row child of parent
                self._close_table(self.stack[-1])
            self._write_suite_header(elem, counts, depth=len(self.stack))
            self.stack.append({"table_open": False})
        elif elem.tag == "testcase":
            self.in_testcase = True

    def handle_end(self, elem):
        # While inside a testcase, ignore its sub-elements (failure/system-out/...)
        # and render the whole case in one go at its own end event.
        if self.in_testcase:
            if elem.tag == "testcase":
                self._write_case(elem)
                elem.clear()
                self.in_testcase = False
            return
        tag = elem.tag
        if tag == "testsuite":
            frame = self.stack.pop()
            self._close_table(frame)
            self.out.write("</section>")
            elem.clear()
        elif tag == "system-out" and self.stack:
            self._close_table(self.stack[-1])
            self._write_output("stdout", _text(elem))
        elif tag == "system-err" and self.stack:
            self._close_table(self.stack[-1])
            self._write_output("stderr", _text(elem))

    # -- writers -----------------------------------------------------------
    def _close_table(self, frame):
        if frame["table_open"]:
            self.out.write(TABLE_CLOSE)
            frame["table_open"] = False

    def _write_suite_header(self, elem, counts, depth):
        lvl = min(2 + depth, 6)
        name = esc(elem.get("name") or "(unnamed suite)")
        pkg = elem.get("package", "")
        time = elem.get("time", "")
        failed = counts.get("failed", 0) + counts.get("error", 0)
        self.out.write(f'<section class="suite mb-4"><h{lvl} class="mb-2">{name}')
        if pkg:
            self.out.write(f" <small>{esc(pkg)}</small>")
        self.out.write(f"</h{lvl}>")
        self.out.write(
            '<div class="mb-2">'
            f'<span class="badge text-bg-dark me-2">{counts.get("total", 0)} tests</span>'
            f'<span class="badge {_count_bg(counts.get("passed", 0), "text-bg-success")} me-2">{counts.get("passed", 0)} passed</span>'
            f'<span class="badge {_count_bg(failed, "text-bg-danger")} me-2">{failed} failed</span>'
            f'<span class="badge {_count_bg(counts.get("skipped", 0), "text-bg-secondary")} me-2">{counts.get("skipped", 0)} skipped</span>'
        )
        if time:
            self.out.write(f'<span class="small">{esc(time)}s</span>')
        self.out.write("</div>")

    def _write_case(self, elem):
        if not self.stack:
            return
        frame = self.stack[-1]
        if not frame["table_open"]:
            self.out.write(TABLE_OPEN)
            frame["table_open"] = True

        status = _classify(elem)
        row_class, badge_class, label = STATUS[status]
        self.out.write(
            f'<tr class="{row_class}">'
            f'<td><span class="badge {badge_class}">{label}</span></td>'
            f'<td>{esc(elem.get("name", ""))}</td>'
            f'<td><code>{esc(elem.get("classname", ""))}</code></td>'
            f'<td>{esc(elem.get("time", ""))}</td></tr>'
        )

        detail = self._case_detail(elem, status)
        if detail:
            self.out.write(f'<tr class="{row_class}"><td colspan="4">{detail}</td></tr>')

    def _case_detail(self, elem, status):
        parts = []
        sub = elem.find("failure")
        if sub is None:
            sub = elem.find("error")
        if sub is None:
            sub = elem.find("skipped")
        if sub is not None:
            message = sub.get("message", "")
            body = _text(sub)
            if message or body:
                open_attr = " open" if status in ("failed", "error") else ""
                summary = esc(message) if message else status
                parts.append(f"<details{open_attr}><summary>{summary}</summary>")
                if body:
                    parts.append(f"<pre>{esc(body)}</pre>")
                parts.append("</details>")
        for label, el in (("stdout", elem.find("system-out")),
                          ("stderr", elem.find("system-err"))):
            text = _text(el)
            if text.strip():
                parts.append(
                    f"<details><summary>{label}</summary><pre>{esc(text)}</pre></details>")
        return "".join(parts)

    def _write_output(self, label, text):
        if text.strip():
            self.out.write(
                f"<details><summary>{label}</summary><pre>{esc(text)}</pre></details>")


def render_summary(out, totals):
    out.write(
        '<div class="card summary-badges">'
        '<h2 class="mb-2">Summary</h2>'
        f'<span class="badge text-bg-dark">Total: {totals.get("total", 0)}</span>'
        f'<span class="badge {_count_bg(totals.get("passed", 0), "text-bg-success")}">Passed: {totals.get("passed", 0)}</span>'
        f'<span class="badge {_count_bg(totals.get("failed", 0), "text-bg-danger")}">Failed: {totals.get("failed", 0)}</span>'
        f'<span class="badge {_count_bg(totals.get("error", 0), "text-bg-warning")}">Errors: {totals.get("error", 0)}</span>'
        f'<span class="badge {_count_bg(totals.get("skipped", 0), "text-bg-secondary")}">Skipped: {totals.get("skipped", 0)}</span>'
        "</div>"
    )


def pass2(valid_files, suite_counts, totals, out, title, bootstrap_url):
    out.write("<!DOCTYPE html>\n")
    out.write('<html lang="en"><head><meta charset="utf-8">')
    out.write('<meta name="viewport" content="width=device-width, initial-scale=1">')
    out.write(f"<title>{esc(title)}</title>")
    out.write(f'<link rel="stylesheet" href="{esc(bootstrap_url)}">')
    out.write(f"<style>{EXTRA_CSS}</style>")
    out.write('</head><body><div class="container-fluid py-4">')
    out.write(f'<h1 class="mb-4">{esc(title)}</h1>')
    render_summary(out, totals)

    r = Renderer(out, suite_counts)
    for path in valid_files:
        # Files were validated in pass 1, so re-parsing here is expected to succeed.
        it = ET.iterparse(path, events=("start", "end"))
        _, root = next(it)
        r.handle_start(root)
        for event, elem in it:
            (r.handle_start if event == "start" else r.handle_end)(elem)

    out.write("</div></body></html>")


def main(argv=None):
    ap = argparse.ArgumentParser(
        description="Render JUnit XML into a single HTML report (streaming).")
    ap.add_argument("input_dir", help="directory searched recursively for *.xml")
    ap.add_argument("output_file", help="HTML file to write")
    ap.add_argument("--title", default="Test Report", help="report title")
    ap.add_argument("--bootstrap-url", default=BOOTSTRAP_URL,
                    help="URL of the Bootstrap CSS referenced by the report")
    args = ap.parse_args(argv)

    in_dir = Path(args.input_dir)
    if not in_dir.is_dir():
        print(f"error: input directory not found: {in_dir}", file=sys.stderr)
        return 2

    files = sorted(in_dir.rglob("*.xml"))
    valid_files, suite_counts, totals = pass1(files)
    if not valid_files:
        print("error: no JUnit reports found", file=sys.stderr)
        return 1

    with open(args.output_file, "w", encoding="utf-8") as out:
        pass2(valid_files, suite_counts, totals, out, args.title, args.bootstrap_url)

    size_mb = Path(args.output_file).stat().st_size / 1e6
    print(f"wrote {args.output_file} "
          f"({size_mb:.2f} MB, {totals.get('total', 0)} tests, "
          f"{len(valid_files)} files, {len(suite_counts)} suites)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
