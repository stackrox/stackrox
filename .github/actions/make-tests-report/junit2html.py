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
    junit2html_streaming.py INPUT_DIR OUTPUT_HTML [--title TITLE]

On CDATA: the content of <system-out>/<system-err> (which JUnit stores as CDATA)
is preserved verbatim and HTML-escaped into <pre>, the HTML equivalent of "treat
this text literally".
"""

import argparse
import html
import sys
import xml.etree.ElementTree as ET
from collections import Counter
from pathlib import Path

BOOTSTRAP_URL = "https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css"

# Small additions on top of Bootstrap (things Bootstrap does not provide).
EXTRA_CSS = (
    ".suite{border-left:3px solid #dee2e6;padding-left:1rem}"
    "pre{white-space:pre-wrap;word-break:break-word}"
    ".summary-badges .badge{font-size:1rem}"
    "summary{cursor:pointer}"
    ".meta{font-size:.875rem}"
    # Self-linking anchors: a '#' that only appears on hover, so the view stays
    # uncluttered. The element keeps its space while hidden to avoid layout shift.
    ".anchor{opacity:0;padding-right:.35rem;text-decoration:none;color:#adb5bd}"
    ".anchor:hover{color:#0d6efd}"
    "h2:hover .anchor,h3:hover .anchor,h4:hover .anchor,h5:hover .anchor,"
    "h6:hover .anchor,.case:hover .anchor{opacity:1}"
    # Highlight whatever a shared #link points at.
    ":target{scroll-margin-top:1rem}"
    "section.suite:target{background:rgba(13,110,253,.06)}"
    ".case:target{outline:2px solid #0d6efd;outline-offset:-2px}"
    # Test cases: a CSS-grid "table" whose rows are <details>, so each case can
    # expand its own output. A fixed column template shared by every row keeps the
    # columns aligned across the otherwise-independent rows. The disclosure caret
    # sits on the test name.
    ".cases-head,.case>summary,.case.nodetail{display:grid;"
    "grid-template-columns:9rem minmax(10rem,2fr) minmax(8rem,3fr) 5rem;"
    "gap:.5rem;align-items:center}"
    ".cases-head{font-weight:600;padding:.4rem .5rem;border-bottom:2px solid #dee2e6}"
    ".case{border-bottom:1px solid #dee2e6}"
    ".case>summary,.case.nodetail{padding:.35rem .5rem}"
    ".case>summary{list-style:none}"
    ".case>summary::-webkit-details-marker{display:none}"
    ".case.s-passed{background:#d1e7dd}"
    ".case.s-failed,.case.s-error{background:#f8d7da}"
    ".case.s-skipped{background:#e2e3e5}"
    ".c-status{white-space:nowrap}"
    ".c-name{padding-left:1.2em;position:relative;word-break:break-word}"
    ".case>summary .c-name::before{content:'▸';position:absolute;left:0;"
    "color:#6c757d}"
    ".case[open]>summary .c-name::before{content:'▾'}"
    ".c-class{word-break:break-all}"
    ".c-time{text-align:right;font-variant-numeric:tabular-nums}"
    ".case-body{padding:.25rem .75rem .6rem 1.7em}"
    ".case-body .lbl{font-weight:600;color:#6c757d;font-size:.85rem;margin-top:.6rem}"
    ".case-body>:first-child{margin-top:0}"
)

# status -> (row class, badge class, label). Errors are treated as failures (red).
STATUS = {
    "passed": ("s-passed", "text-bg-success", "TEST PASSED"),
    "failed": ("s-failed", "text-bg-danger", "TEST FAILED"),
    "error": ("s-error", "text-bg-danger", "TEST ERROR"),
    "skipped": ("s-skipped", "text-bg-secondary", "TEST SKIPPED"),
}

# A suite's cases are a CSS-grid list (not a <table>) so each row can be a
# <details>. LIST_OPEN emits the container and the column-label header row.
LIST_OPEN = ('<div class="cases"><div class="cases-head">'
             '<span>Status</span><span class="c-name">Test</span>'
             '<span>Class</span><span class="c-time">Time (s)</span></div>')
LIST_CLOSE = "</div>"

# One delegated listener for every anchor link. Clicking an anchor that sits
# inside a <summary> would otherwise also toggle its <details>; preventDefault()
# cancels that, and we set the hash ourselves so the address bar / :target still
# update. Done once here instead of an inline onclick on each of thousands of rows.
ANCHOR_SCRIPT = (
    "<script>document.addEventListener('click',function(e){"
    "var a=e.target.closest('a.anchor');"
    "if(a){e.preventDefault();location.hash=a.getAttribute('href');}});</script>"
)

esc = html.escape


def _anchor(anchor_id):
    """A hover-revealed '#' link that points to its own element's id, so readers
    can copy a deep link from the address bar. A single delegated click handler
    (see ANCHOR_SCRIPT) stops an anchor inside a <summary> from also toggling the
    <details> — cheaper than an inline onclick on every one of thousands of rows.

    The visible "#" is the accessible name; we deliberately omit an aria-label
    because it repeats across ~19k rows and ~0.5 MB matters for the size budget.
    """
    return f'<a class="anchor" href="#{anchor_id}">#</a>'


def _count_bg(count, color_class):
    """Bootstrap text-bg class for a count badge: neutral (light) when the count
    is zero, coloured only when it is non-zero."""
    return color_class if count else "text-bg-light"


def _suite_attr_rows(elem):
    """(label, value) rows for the <testsuite> attributes shown in the Meta block.
    Empty ones are dropped later by render_metadata."""
    return [
        ("Timestamp", elem.get("timestamp", "")),
        ("Assertions", elem.get("assertions", "")),
        ("Hostname", elem.get("hostname", "")),
        ("File", elem.get("file", "")),
        ("ID", elem.get("id", "")),
    ]


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
    suites = []  # aggregated Counter per suite, in start order
    suite_io = []  # {"stdout":.., "stderr":..} per suite, in start order
    stack = []  # (counter, io) for currently open suites
    file_totals = Counter()
    depth_tc = 0  # testcase nesting; only suite-direct output is captured

    it = ET.iterparse(path, events=("start", "end"))
    try:
        _, root = next(it)
    except StopIteration:
        raise ET.ParseError("empty document")
    if root.tag not in ("testsuites", "testsuite"):
        raise NotJUnit(root.tag)

    def on_start(elem):
        nonlocal depth_tc
        if elem.tag == "testsuite":
            c, io = Counter(), {}
            suites.append(c)
            suite_io.append(io)
            stack.append((c, io))
        elif elem.tag == "testcase":
            depth_tc += 1

    def on_end(elem):
        nonlocal depth_tc
        if elem.tag == "testcase":
            if stack:
                stack[-1][0]["total"] += 1
                stack[-1][0][_classify(elem)] += 1
            depth_tc -= 1
            elem.clear()
        elif elem.tag == "testsuite":
            c, _ = stack.pop()
            (stack[-1][0] if stack else file_totals).update(c)
            elem.clear()
        elif elem.tag in ("system-out", "system-err") and not depth_tc and stack:
            # Suite-level output appears after the cases; capture it here so pass 2
            # can render it before the table. Testcase output (depth_tc>0) is left
            # for pass 2's per-case rendering.
            text = _text(elem)
            if text.strip():
                key = "stdout" if elem.tag == "system-out" else "stderr"
                stack[-1][1][key] = text
            elem.clear()

    on_start(root)
    for event, elem in it:
        (on_start if event == "start" else on_end)(elem)
    return suites, suite_io, file_totals


def pass1(files):
    valid_files, suite_counts, suite_io, totals = [], [], [], Counter()
    for path in files:
        try:
            suites, ios, file_totals = count_file(path)
        except NotJUnit as e:
            print(f"{path}: not a JUnit report (root <{e.tag}>); skipping",
                  file=sys.stderr)
            continue
        except ET.ParseError as e:
            print(f"{path}: not well-formed XML ({e}); skipping", file=sys.stderr)
            continue
        valid_files.append(path)
        suite_counts.extend(suites)  # global order == pass-2 suite-start order
        suite_io.extend(ios)  # parallel to suite_counts
        totals.update(file_totals)
    return valid_files, suite_counts, suite_io, totals


# --------------------------------------------------------------------------- #
# Pass 2: render, streaming to the output file object.
# --------------------------------------------------------------------------- #
class Renderer:
    def __init__(self, out, suite_counts, suite_io):
        self.out = out
        self.suite_counts = suite_counts
        self.suite_io = suite_io
        self.sid = 0  # next suite id (matches pass-1 start order)
        self.stack = []  # frames: {"list_open": bool, "preamble_written": ...}
        self.in_testcase = False
        self.suite_no = 0  # per-document anchor counters
        self.case_no = 0

    def handle_start(self, elem):
        if elem.tag == "testsuite":
            counts = self.suite_counts[self.sid]
            io = self.suite_io[self.sid]
            self.sid += 1
            if self.stack:  # a nested suite is a non-row child of parent
                self._flush_preamble(self.stack[-1])
                self._close_list(self.stack[-1])
            self._write_suite_header(elem, counts, depth=len(self.stack))
            # Attributes are known now; <property> children stream in before the
            # cases and get appended to "props". Suite-level stdout/stderr were
            # captured in pass 1 (they follow the cases in the XML). The preamble
            # renders Meta then output, once, before the table.
            self.stack.append({
                "list_open": False,
                "preamble_written": False,
                "attr_rows": _suite_attr_rows(elem),
                "props": [],
                "io": io,
            })
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
            # A suite with only Meta/output and no cases still gets its preamble.
            self._flush_preamble(frame)
            self._close_list(frame)
            self.out.write("</section>")
            elem.clear()
        elif tag == "property" and self.stack:
            # Suite-level <property>; case-level ones are skipped via in_testcase.
            value = elem.get("value")
            if value is None:
                value = _text(elem)
            self.stack[-1]["props"].append((elem.get("name", ""), (value or "").strip()))
            elem.clear()

    # -- writers -----------------------------------------------------------
    def _close_list(self, frame):
        if frame["list_open"]:
            self.out.write(LIST_CLOSE)
            frame["list_open"] = False

    def _flush_preamble(self, frame):
        """Render, once and before any table, the suite's Meta block followed by
        its suite-level stdout/stderr."""
        if frame["preamble_written"]:
            return
        frame["preamble_written"] = True
        render_metadata(self.out, frame["attr_rows"] + frame["props"])
        io = frame["io"]
        self._write_output("stdout", io.get("stdout", ""))
        self._write_output("stderr", io.get("stderr", ""))

    def _write_suite_header(self, elem, counts, depth):
        lvl = min(2 + depth, 6)
        name = esc(elem.get("name") or "(unnamed suite)")
        pkg = elem.get("package", "")
        time = elem.get("time", "")
        failed = counts.get("failed", 0) + counts.get("error", 0)
        self.suite_no += 1
        anchor_id = f"suite-{self.suite_no}"
        self.out.write(
            f'<section class="suite mb-4" id="{anchor_id}">'
            f'<h{lvl} class="mb-2">{_anchor(anchor_id)}{name}')
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
            self.out.write(f'<span class="small">{esc(time)}</span>')
        self.out.write("</div>")

    def _write_case(self, elem):
        if not self.stack:
            return
        frame = self.stack[-1]
        self._flush_preamble(frame)  # Meta + suite output go before the cases list
        if not frame["list_open"]:
            self.out.write(LIST_OPEN)
            frame["list_open"] = True

        status = _classify(elem)
        row_class, badge_class, label = STATUS[status]
        self.case_no += 1
        anchor_id = f"test-{self.case_no}"
        cells = (
            f'<span class="c-status">{_anchor(anchor_id)}'
            f'<span class="badge {badge_class}">{label}</span></span>'
            f'<span class="c-name">{esc(elem.get("name", ""))}</span>'
            f'<span class="c-class"><code>{esc(elem.get("classname", ""))}</code></span>'
            f'<span class="c-time">{esc(elem.get("time", ""))}</span>'
        )
        body = self._case_body(elem)
        if body:
            # The case is a <details> (class "io" -> toggled by Expand/Collapse
            # all). Collapsed by default; the caret sits on the test name.
            self.out.write(
                f'<details class="io case {row_class}" id="{anchor_id}">'
                f"<summary>{cells}</summary>"
                f'<div class="case-body">{body}</div></details>'
            )
        else:
            self.out.write(
                f'<div class="case nodetail {row_class}" id="{anchor_id}">{cells}</div>'
            )

    def _case_body(self, elem):
        """Inner detail of a case: failure/error/skipped message and trace plus
        stdout/stderr, rendered directly. The enclosing case <details> is the only
        expand/collapse control, so there are no nested toggles here."""
        parts = []
        sub = elem.find("failure")
        if sub is None:
            sub = elem.find("error")
        if sub is None:
            sub = elem.find("skipped")
        if sub is not None:
            message = sub.get("message", "")
            body = _text(sub)
            if message:
                parts.append(f'<div class="lbl">{esc(message)}</div>')
            if body:
                parts.append(f"<pre>{esc(body)}</pre>")
        for label, el in (("stdout", elem.find("system-out")),
                          ("stderr", elem.find("system-err"))):
            text = _text(el)
            if text.strip():
                parts.append(f'<div class="lbl">{label}</div><pre>{esc(text)}</pre>')
        return "".join(parts)

    def _write_output(self, label, text):
        if text.strip():
            self.out.write(
                f'<details class="io"><summary>{label}</summary><pre>{esc(text)}</pre></details>')


def render_metadata(out, rows, summary="Meta"):
    """Render a collapsed, hover-expandable block of key/value metadata as a
    headerless two-column table in a slightly smaller font.

    `rows` is an iterable of (key, value) pairs; empty values are dropped. The
    block is tagged 'io' so the global Expand all / Collapse all buttons toggle
    it. Reusable: suites use it now, the summary can use it later.
    """
    rows = [(k, v) for k, v in rows if v]
    if not rows:
        return
    out.write(f'<details class="io meta"><summary>{esc(summary)}</summary>')
    out.write('<table class="table table-sm table-borderless w-auto mb-0"><tbody>')
    for k, v in rows:
        out.write(f'<tr><td class="text-body-secondary pe-3">{esc(k)}</td>'
                  f"<td>{esc(v)}</td></tr>")
    out.write("</tbody></table></details>")


def render_summary(out, totals):
    # Summary card with the count chips, followed by the expand/collapse controls.
    # The buttons toggle every collapsible block tagged <details class="io">:
    # stdout/stderr and failure/error/skipped detail blocks. Plain JS, no libraries.
    out.write(
        '<div class="card shadow-sm mb-3"><div class="card-body">'
        '<h2 class="h6 text-uppercase text-body-secondary mb-3">Summary</h2>'
        '<div class="d-flex flex-wrap gap-2 summary-badges">'
        f'<span class="badge text-bg-dark">{totals.get("total", 0)} tests</span>'
        f'<span class="badge {_count_bg(totals.get("passed", 0), "text-bg-success")}">{totals.get("passed", 0)} passed</span>'
        f'<span class="badge {_count_bg(totals.get("failed", 0), "text-bg-danger")}">{totals.get("failed", 0)} failed</span>'
        f'<span class="badge {_count_bg(totals.get("error", 0), "text-bg-warning")}">{totals.get("error", 0)} errored</span>'
        f'<span class="badge {_count_bg(totals.get("skipped", 0), "text-bg-secondary")}">{totals.get("skipped", 0)} skipped</span>'
        "</div></div></div>"
    )


def render_buttons(out):
    out.write(
        '<div class="btn-group mb-4" role="group" aria-label="Expand or collapse all">'
        '<button type="button" class="btn btn-sm btn-outline-secondary" '
        "onclick=\"document.querySelectorAll('details.io').forEach(d => d.open = true)\">"
        "Expand all</button>"
        '<button type="button" class="btn btn-sm btn-outline-secondary" '
        "onclick=\"document.querySelectorAll('details.io').forEach(d => d.open = false)\">"
        "Collapse all</button>"
        "</div>"
    )


def pass2(valid_files, suite_counts, suite_io, totals, out, title):
    out.write("<!DOCTYPE html>\n")
    out.write('<html lang="en"><head><meta charset="utf-8">')
    out.write('<meta name="viewport" content="width=device-width, initial-scale=1">')
    out.write(f"<title>{esc(title)}</title>")
    out.write(f'<link rel="stylesheet" href="{esc(BOOTSTRAP_URL)}">')
    out.write(f"<style>{EXTRA_CSS}</style>")
    out.write('</head><body><div class="container-fluid py-4">')
    out.write(f'<h1 class="mb-4">{esc(title)}</h1>')
    render_summary(out, totals)
    render_buttons(out)

    r = Renderer(out, suite_counts, suite_io)
    for path in valid_files:
        # Files were validated in pass 1, so re-parsing here is expected to succeed.
        it = ET.iterparse(path, events=("start", "end"))
        _, root = next(it)
        r.handle_start(root)
        for event, elem in it:
            (r.handle_start if event == "start" else r.handle_end)(elem)

    out.write("</div>")
    out.write(ANCHOR_SCRIPT)
    out.write("</body></html>")


def main(argv=None):
    ap = argparse.ArgumentParser(
        description="Render JUnit XML into a single HTML report (streaming).")
    ap.add_argument("input_dir", help="directory searched recursively for *.xml")
    ap.add_argument("output_file", help="HTML file to write")
    ap.add_argument("--title", default="Test Report", help="report title")
    args = ap.parse_args(argv)

    in_dir = Path(args.input_dir)
    if not in_dir.is_dir():
        print(f"error: input directory not found: {in_dir}", file=sys.stderr)
        return 2

    files = sorted(in_dir.rglob("*.xml"))
    valid_files, suite_counts, suite_io, totals = pass1(files)
    if not valid_files:
        print("error: no JUnit reports found", file=sys.stderr)
        return 1

    with open(args.output_file, "w", encoding="utf-8") as out:
        pass2(valid_files, suite_counts, suite_io, totals, out, args.title)

    size_mb = Path(args.output_file).stat().st_size / 1024.0 / 1024.0
    print(f"wrote {args.output_file} "
          f"({size_mb:.2f} MiB, {totals.get('total', 0)} tests, "
          f"{len(valid_files)} files, {len(suite_counts)} suites)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
