#!/usr/bin/env python3
"""junit2html.py - convert a tree of JUnit XML files into one self-contained HTML report.

Implements the v3 technical specification. Standard library only. Streaming
(two-pass) design using xml.etree.ElementTree.iterparse.
"""

import argparse
import html
import os
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

BOOTSTRAP_CSS = "https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def e(s):
    """HTML-escape text content."""
    return html.escape(s or "", quote=False)


def text_of(el):
    """Full character content (text + CDATA) of an element."""
    return "".join(el.itertext())


# status class -> (label, badge color class)
STATUS_LABEL = {
    "passed": ("PASSED", "text-bg-success"),
    "failed": ("TEST FAILED", "text-bg-danger"),
    "error": ("TEST ERROR", "text-bg-danger"),
    "skipped": ("TEST SKIPPED", "text-bg-secondary"),
}
# outcome class -> row background class
ROW_CLASS = {"passed": "p", "failed": "f", "error": "f", "skipped": "s"}
# class -> index into [passed, failed, error, skipped]
CLASS_IDX = {"passed": 0, "failed": 1, "error": 2, "skipped": 3}

STATUS_OVERRIDE = {
    "failed": "failed", "failure": "failed", "fail": "failed",
    "broken": "failed", "regression": "failed",
    "error": "error", "errored": "error",
    "skipped": "skipped", "skip": "skipped", "disabled": "skipped",
    "ignored": "skipped", "notrun": "skipped", "not_run": "skipped",
}


def classify(tc):
    """Return (class, marker_element_or_None) for a <testcase> element."""
    f = tc.find("failure")
    if f is not None:
        return "failed", f
    er = tc.find("error")
    if er is not None:
        return "error", er
    sk = tc.find("skipped")
    if sk is not None:
        return "skipped", sk
    status = (tc.get("status") or "").strip().lower()
    cls = STATUS_OVERRIDE.get(status)
    if cls:
        return cls, None
    return "passed", None


def new_rec(el):
    return {
        "name": el.get("name") or "",
        "package": el.get("package") or "",
        "time": el.get("time") or "",
        "timestamp": el.get("timestamp") or "",
        "assertions": el.get("assertions") or "",
        "hostname": el.get("hostname") or "",
        "file": el.get("file") or "",
        "id": el.get("id") or "",
        "props": [],
        "sysout": "",
        "syserr": "",
        "counts": [0, 0, 0, 0],  # passed, failed, error, skipped
    }


def extract_props(props_el):
    out = []
    for p in props_el.findall("property"):
        name = p.get("name") or ""
        val = p.get("value")
        if val is None:
            val = text_of(p)
        val = (val or "").strip()
        if val:
            out.append((name, val))
    return out


# ---------------------------------------------------------------------------
# Pass 1: validate + count + capture suite-level meta/output
# ---------------------------------------------------------------------------

def parse_pass1(path):
    """Fully stream-parse one file.

    Returns a tuple (status, payload):
      ("ok", (suites_list, totals))         - valid
      ("bad_root", tag)                     - wrong root element
      ("empty", None)                       - no root element
      ("malformed", detail)                 - not well-formed XML
    """
    file_suites = []
    suite_stack = []
    tag_stack = []
    tc_depth = 0
    totals = [0, 0, 0, 0]
    root_tag = None
    try:
        for ev, el in ET.iterparse(str(path), events=("start", "end")):
            tag = el.tag
            if ev == "start":
                if root_tag is None:
                    root_tag = tag
                    if tag not in ("testsuites", "testsuite"):
                        return ("bad_root", tag)
                tag_stack.append(tag)
                if tag == "testsuite":
                    rec = new_rec(el)
                    file_suites.append(rec)
                    suite_stack.append(rec)
                elif tag == "testcase":
                    tc_depth += 1
            else:  # end
                parent = tag_stack[-2] if len(tag_stack) >= 2 else None
                if tag == "testcase":
                    cls, _ = classify(el)
                    idx = CLASS_IDX[cls]
                    totals[idx] += 1
                    for rec in suite_stack:
                        rec["counts"][idx] += 1
                    tc_depth -= 1
                    el.clear()
                elif tag == "properties":
                    if tc_depth == 0 and parent == "testsuite" and suite_stack:
                        suite_stack[-1]["props"].extend(extract_props(el))
                elif tag == "system-out":
                    if tc_depth == 0 and parent == "testsuite" and suite_stack:
                        suite_stack[-1]["sysout"] += text_of(el)
                elif tag == "system-err":
                    if tc_depth == 0 and parent == "testsuite" and suite_stack:
                        suite_stack[-1]["syserr"] += text_of(el)
                elif tag == "testsuite":
                    if suite_stack:
                        suite_stack.pop()
                    el.clear()
                if tag_stack:
                    tag_stack.pop()
        return ("ok", (file_suites, totals))
    except ET.ParseError as exc:
        if root_tag is None:
            msg = str(exc)
            if msg.startswith("no element found"):
                return ("empty", None)
            return ("malformed", msg)
        return ("malformed", str(exc))


# ---------------------------------------------------------------------------
# Rendering helpers (pass 2)
# ---------------------------------------------------------------------------

def badge(n, color_nonzero, label):
    cls = color_nonzero if n > 0 else "text-bg-light"
    return f'<span class="badge {cls} me-1">{n} {label}</span>'


def kv_table(rows):
    """Render a headerless two-column table. rows: list of (key, value)."""
    if not rows:
        return ""
    parts = ['<table class="table table-sm table-borderless w-auto kv">', "<tbody>"]
    for k, v in rows:
        parts.append(f"<tr><td>{e(k)}</td><td>{e(v)}</td></tr>")
    parts.append("</tbody></table>")
    return "".join(parts)


def suite_header(out, rec, seq, depth):
    level = min(2 + depth, 6)
    name = rec["name"].strip() or "(unnamed suite)"
    pkg = rec["package"].strip()
    pkg_html = f' <small class="text-muted">{e(pkg)}</small>' if pkg else ""
    out.write(f'<section id="s{seq}" class="suite">')
    out.write(
        f'<h{level} class="sh">{e(name)}{pkg_html}'
        f'<a class="anchor" href="#s{seq}">#</a></h{level}>'
    )
    # badges
    c = rec["counts"]
    total = c[0] + c[1] + c[2] + c[3]
    out.write('<div class="mb-2">')
    out.write(f'<span class="badge text-bg-dark me-1">{total} tests</span>')
    out.write(badge(c[0], "text-bg-success", "passed"))
    out.write(badge(c[1] + c[2], "text-bg-danger", "failed"))
    out.write(badge(c[3], "text-bg-secondary", "skipped"))
    if rec["time"].strip():
        out.write(f'<small class="text-muted">{e(rec["time"].strip())}</small>')
    out.write("</div>")
    # Meta block
    meta_rows = []
    for label, key in (("Timestamp", "timestamp"), ("Assertions", "assertions"),
                       ("Hostname", "hostname"), ("File", "file"), ("ID", "id")):
        val = rec[key].strip()
        if val:
            meta_rows.append((label, val))
    for pn, pv in rec["props"]:
        meta_rows.append((pn, pv))
    if meta_rows:
        out.write('<details class="io mb-2"><summary>Meta</summary>')
        out.write(kv_table(meta_rows))
        out.write("</details>")
    # suite-level system-out / system-err
    for label, text in (("system-out", rec["sysout"]), ("system-err", rec["syserr"])):
        t = text.strip()
        if t:
            out.write(
                f'<details class="io mb-2"><summary>{label}</summary>'
                f"<pre>{e(t)}</pre></details>"
            )


def case_header_row(out):
    out.write(
        '<div class="row-grid hdr"><div>Status</div><div>Test</div>'
        "<div>Time (s)</div></div>"
    )


def render_case(out, tc, seq, suite_name):
    cls, marker = classify(tc)
    label, color = STATUS_LABEL[cls]
    name = tc.get("name") or ""
    time = tc.get("time") or ""
    classname = tc.get("classname") or ""

    # key/value table rows
    kv_rows = []
    if classname and classname != suite_name:
        kv_rows.append(("Class", classname))
    for lbl, attr in (("File", "file"), ("Line", "line"), ("Assertions", "assertions")):
        v = (tc.get(attr) or "").strip()
        if v:
            kv_rows.append((lbl, v))
    if marker is not None:
        mtype = (marker.get("type") or "").strip()
        if mtype:
            kv_rows.append(("Type", mtype))
    for props_el in tc.findall("properties"):
        for pn, pv in extract_props(props_el):
            kv_rows.append((pn, pv))

    # marker message + body
    msg = ""
    trace = ""
    if marker is not None:
        msg = (marker.get("message") or "").strip()
        trace = text_of(marker).strip()

    # case system-out / system-err
    sysout = "".join(text_of(x) for x in tc.findall("system-out")).strip()
    syserr = "".join(text_of(x) for x in tc.findall("system-err")).strip()

    body_parts = []
    kv_html = kv_table(kv_rows)
    if kv_html:
        body_parts.append(kv_html)
    if msg:
        body_parts.append(f"<div><strong>{e(msg)}</strong></div>")
    if trace:
        body_parts.append(f"<pre>{e(trace)}</pre>")
    if sysout:
        body_parts.append(f'<div class="fw-semibold small">system-out</div><pre>{e(sysout)}</pre>')
    if syserr:
        body_parts.append(f'<div class="fw-semibold small">system-err</div><pre>{e(syserr)}</pre>')

    rowbg = ROW_CLASS[cls]
    status_cell = (
        f'<div><span class="badge {color}">{label}</span>'
        f'<a class="anchor" href="#t{seq}">#</a></div>'
    )
    time_cell = f"<div>{e(time)}</div>"

    if body_parts:
        name_cell = f"<div>{e(name)}</div>"
        out.write(f'<details class="io case" id="t{seq}">')
        out.write(f'<summary class="row-grid {rowbg}">{status_cell}{name_cell}{time_cell}</summary>')
        out.write(f'<div class="cbody">{"".join(body_parts)}</div>')
        out.write("</details>")
    else:
        name_cell = f"<div>{e(name)}</div>"
        out.write(
            f'<div class="row-grid {rowbg}" id="t{seq}">'
            f"{status_cell}{name_cell}{time_cell}</div>"
        )


# ---------------------------------------------------------------------------
# HTML scaffolding
# ---------------------------------------------------------------------------

EXTRA_CSS = """
body{padding:1rem}
h1{margin-bottom:1rem}
.anchor{visibility:hidden;margin-left:.4rem;text-decoration:none;color:#6c757d;font-weight:normal}
.sh:hover .anchor,.row-grid:hover .anchor{visibility:visible}
.row-grid{display:grid;grid-template-columns:9rem minmax(10rem,1fr) 5rem;align-items:center;border-bottom:1px solid rgba(0,0,0,.15)}
.row-grid>*{padding:.25rem .5rem;overflow-wrap:anywhere;min-width:0}
.row-grid.hdr{font-weight:600;background:#f8f9fa}
.p{background:#d1e7dd}.f{background:#f8d7da}.s{background:#e2e3e5}
details.case>summary{list-style:none;cursor:pointer}
details.case>summary::-webkit-details-marker{display:none}
details.case>summary>:nth-child(2)::before{content:"\\25B8  ";color:#6c757d}
details.case[open]>summary>:nth-child(2)::before{content:"\\25BE  ";color:#6c757d}
.cbody{padding:.5rem 1rem;background:#fff}
.kv{font-size:.85rem;margin-bottom:.5rem}
pre{white-space:pre-wrap;word-break:break-word;background:#f8f9fa;padding:.5rem;border-radius:.25rem;margin-bottom:.5rem}
section{margin-bottom:1.5rem}
section:target{background:#fff3cd}
section.suite{border-left: 3px solid #dee2e6;padding-left: 1rem;}
details.case:target,.row-grid:target{outline:2px solid #0d6efd}
summary{cursor:pointer}
"""

SCRIPT = """
document.addEventListener('click',function(ev){
  var a=ev.target.closest('a.anchor');
  if(a){ev.preventDefault();location.hash=a.getAttribute('href');}
});
"""


def write_head(out, title):
    out.write("<!doctype html><html lang=\"en\"><head>")
    out.write('<meta charset="utf-8">')
    out.write('<meta name="viewport" content="width=device-width, initial-scale=1">')
    out.write(f"<title>{e(title)}</title>")
    out.write(f'<link rel="stylesheet" href="{BOOTSTRAP_CSS}">')
    out.write(f"<style>{EXTRA_CSS}</style>")
    out.write("</head><body>")
    out.write('<div class="container-fluid">')
    out.write(f"<h1>{e(title)}</h1>")


def write_summary(out, totals):
    passed, failed, error, skipped = totals
    total = passed + failed + error + skipped
    out.write('<div class="card mb-3"><div class="card-body">')
    out.write('<h5 class="card-title">Summary</h5>')
    out.write(f'<span class="badge text-bg-dark me-1">{total} tests</span>')
    out.write(badge(passed, "text-bg-success", "passed"))
    out.write(badge(failed, "text-bg-danger", "failed"))
    out.write(badge(error, "text-bg-warning", "errored"))
    out.write(badge(skipped, "text-bg-secondary", "skipped"))
    out.write("</div></div>")


def write_controls(out):
    out.write('<div class="mb-3">')
    out.write('<button class="btn btn-sm btn-outline-secondary me-2" '
              "onclick=\"document.querySelectorAll('details.io').forEach(function(d){d.open=true;})\">"
              "Expand all</button>")
    out.write('<button class="btn btn-sm btn-outline-secondary" '
              "onclick=\"document.querySelectorAll('details.io').forEach(function(d){d.open=false;})\">"
              "Collapse all</button>")
    out.write("</div>")


# ---------------------------------------------------------------------------
# Pass 2: render
# ---------------------------------------------------------------------------

def render_pass2(out, valid_files, suites, totals, title):
    write_head(out, title)
    write_summary(out, totals)
    write_controls(out)

    suite_idx = 0  # index into global `suites`
    suite_seq = 0  # 1-based s<N>
    case_seq = 0  # 1-based t<N>

    for path in valid_files:
        depth_stack = []  # records of currently-open suites
        tag_stack = []
        for ev, el in ET.iterparse(str(path), events=("start", "end")):
            tag = el.tag
            if ev == "start":
                tag_stack.append(tag)
                if tag == "testsuite":
                    rec = suites[suite_idx]
                    suite_idx += 1
                    suite_seq += 1
                    depth = len(depth_stack)
                    suite_header(out, rec, suite_seq, depth)
                    rec["_hdr_done"] = False
                    depth_stack.append(rec)
            else:
                if tag == "testcase":
                    case_seq += 1
                    owner = depth_stack[-1] if depth_stack else None
                    suite_name = owner["name"] if owner else ""
                    if owner is not None and not owner.get("_hdr_done"):
                        case_header_row(out)
                        owner["_hdr_done"] = True
                    render_case(out, el, case_seq, suite_name)
                    el.clear()
                elif tag == "testsuite":
                    out.write("</section>")
                    if depth_stack:
                        depth_stack.pop()
                    el.clear()
                if tag_stack:
                    tag_stack.pop()

    out.write("</div>")  # container-fluid
    out.write(f"<script>{SCRIPT}</script>")
    out.write("</body></html>")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main(argv=None):
    parser = argparse.ArgumentParser(add_help=True)
    parser.add_argument("input_dir")
    parser.add_argument("output_html")
    parser.add_argument("--title", default="Test Report")
    args = parser.parse_args(argv)

    input_dir = args.input_dir
    if not os.path.isdir(input_dir):
        sys.stderr.write(f"error: input directory not found: {input_dir}\n")
        return 2

    files = sorted(Path(input_dir).rglob("*.xml"), key=lambda p: str(p))

    # Pass 1
    valid_files = []
    suites = []
    totals = [0, 0, 0, 0]
    for path in files:
        status, payload = parse_pass1(path)
        if status == "ok":
            file_suites, file_totals = payload
            suites.extend(file_suites)
            for i in range(4):
                totals[i] += file_totals[i]
            valid_files.append(path)
            sys.stdout.write(f"processing {path}\n")
        elif status == "bad_root":
            sys.stderr.write(f"{path}: not a JUnit report (root {payload}); skipping\n")
        elif status == "empty":
            sys.stderr.write(f"{path}: not well-formed XML (empty document); skipping\n")
        else:  # malformed
            sys.stderr.write(f"{path}: not well-formed XML ({payload}); skipping\n")

    if not valid_files:
        sys.stderr.write("error: no JUnit reports found\n")
        return 1

    # Pass 2
    with open(args.output_html, "w", encoding="utf-8") as out:
        render_pass2(out, valid_files, suites, totals, args.title)

    size = os.path.getsize(args.output_html)
    mib = size / 1024 / 1024
    n_tests = totals[0] + totals[1] + totals[2] + totals[3]
    sys.stdout.write(
        f"wrote {args.output_html} ({mib:.2f} MiB, {n_tests} tests, "
        f"{len(valid_files)} files, {len(suites)} suites)\n"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
