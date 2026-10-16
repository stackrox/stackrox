#!/usr/bin/env python3

import argparse
import html
import sys
import tempfile
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from enum import Enum
from pathlib import Path

BOOTSTRAP_CSS_URL = "https://cdn.jsdelivr.net/npm/bootstrap@5.3.8/dist/css/bootstrap.min.css"


def main():
    args = parse_args()

    input_dir = Path(args.input_dir)
    input_dir.stat()  # Raise error if input_dir does not exist.

    xml_files = sorted(input_dir.rglob('*.xml'), key=lambda p: str(p))
    print(f'Found XML files: {len(xml_files)}')

    grand_totals = make_counts()
    valid_files = []

    for f in xml_files:
        print(f'Parsing {f} ... ', end='')
        ok, suites = pass1_parse_file(f)
        if not ok:
            print('not a JUnit XML file')
            continue
        file_totals = make_counts()
        for s in suites:
            sum_up(grand_totals, s.counts)
            sum_up(file_totals, s.counts)
        print(f'found {file_totals[Ctr.TESTS]} tests')
        valid_files.append((f, suites))

    if not valid_files:
        raise RuntimeError('No valid JUnit reports found')

    output_file = Path(args.output_file)

    with tempfile.NamedTemporaryFile(mode='wt', encoding='utf-8', dir=output_file.parent, delete=False,
                                     prefix=output_file.name + '-tmp-') as tmp_out:
        tmp_name = tmp_out.name
        pass2_render(tmp_out, args.title, grand_totals, valid_files)

    Path(tmp_name).rename(output_file)

    size_mib = output_file.stat().st_size / 1024.0 / 1024
    print(f'Wrote {output_file} ({size_mib:.2f} MiB, {len(valid_files)} input files, ' +
          f'{grand_totals[Ctr.TESTS]} tests, {grand_totals[Ctr.FAILURES]} failed tests, ' +
          f'{grand_totals[Ctr.SKIPPED]} skipped tests, {grand_totals[Ctr.ERRORS]} test errors)')

    return 0


def parse_args():
    parser = argparse.ArgumentParser(add_help=True)
    parser.add_argument('input_dir', metavar='input-dir', help='Input directory with JUnit XML files.')
    parser.add_argument('output_file', metavar='output-file', help='Output HTML report file.')
    parser.add_argument('--title', default='Test Report', help='Title to put in the report file.')
    return parser.parse_args(sys.argv[1:])


class Ctr(Enum):
    """Counter"""
    TESTS = 1
    PASSED = 2
    FAILURES = 3
    ERRORS = 4
    SKIPPED = 5


def make_counts():
    return {
        Ctr.TESTS: 0,
        Ctr.PASSED: 0,
        Ctr.FAILURES: 0,
        Ctr.ERRORS: 0,
        Ctr.SKIPPED: 0,
    }


def sum_up(acc, item):
    for k in acc.keys():
        acc[k] += item[k]


@dataclass
class TestSuite:
    name: str
    counts: dict[Ctr, int]
    time: str
    props: list[tuple[str, str]]
    sysout: str
    syserr: str


def pass1_parse_file(file):
    suites = []
    root_tag = None
    test_suite = None
    test_case = None
    try:
        for event, elem in ET.iterparse(file, events=('start', 'end')):
            tag = elem.tag
            if event == 'start':
                if root_tag is None:
                    root_tag = tag
                    if tag not in ('testsuites', 'testsuite'):
                        return False, []
                if tag == 'testsuite':
                    ensure(test_suite is None, 'nested testsuite-s are not supported')
                    test_suite = make_test_suite(elem)
                    suites.append(test_suite)
                elif tag == 'testcase':
                    ensure(test_suite is not None, 'error: testcase outside testsuite')
                    ensure(test_case is None, 'nested testcase-s are not supported')
                    test_case = elem
            else:  # event == 'end'
                if tag == 'testsuite':
                    ensure(test_suite.counts[Ctr.TESTS] * 2 == sum(test_suite.counts.values()),
                           'something is wrong with our tests counting')
                    test_suite = None
                elif tag == 'testcase':
                    test_case = None
                    test_suite.counts[Ctr.TESTS] += 1
                    outcome = classify(elem)
                    test_suite.counts[outcome] += 1
                elif tag == 'system-out':
                    if test_case is None:
                        ensure(not test_suite.sysout, 'multiple system-out elements are not supported')
                        test_suite.sysout = text_of(elem)
                elif tag == 'system-err':
                    if test_case is None:
                        ensure(not test_suite.syserr, 'multiple system-err elements are not supported')
                        test_suite.syserr = text_of(elem)
                elif tag == 'properties':
                    if test_case is None:
                        test_suite.props.extend(extract_props(elem))
        return True, suites
    except ET.ParseError:
        return False, []


def ensure(cond, message):
    assert isinstance(cond, bool)
    if not cond:
        raise RuntimeError(message)


def make_test_suite(el):
    props = []
    for k, v in el.items():
        if k in {'name', 'time'}:
            continue
        props.append((k, v))

    return TestSuite(
        name=el.get('name', ''),
        counts=make_counts(),
        time=el.get('time', ''),
        props=props,
        sysout='',
        syserr='',
    )


def text_of(el):
    """Full character content (text + CDATA) of an element."""
    return ''.join(el.itertext())


def extract_props(el):
    out = []
    for p in el.findall('property'):
        name = p.get('name', '')
        val = (p.get('value', '') or text_of(p)).strip()
        out.append((name, val))
    return out


def classify(testcase_el):
    if testcase_el.find('failure') is not None:
        return Ctr.FAILURES
    elif testcase_el.find('error') is not None:
        return Ctr.ERRORS
    elif testcase_el.find('skipped') is not None:
        return Ctr.SKIPPED
    ensure(not testcase_el.get('status'), 'status override attribute is not supported')
    return Ctr.PASSED


def pass2_render(out, title, grand_totals, valid_files):
    write_doc_start(out, title)
    write_summary(out, grand_totals)
    write_controls(out)

    href_suite_counter = Counter()
    for file, suites in valid_files:
        pass2_render_file(out, file, suites, href_suite_counter)

    write_doc_end(out)


def write_doc_start(out, title):
    extra_css = """
        body { padding: 1rem }
        h1 { margin-bottom: 1rem }
        .anchor {
            visibility: hidden;
            margin-left: .4rem;
            text-decoration: none;
            color: #6c757d;
            font-weight: normal
        }
        .sh:hover .anchor, .row-grid:hover .anchor { visibility: visible }
        .row-grid {
            display: grid;
            grid-template-columns:9rem minmax(10rem, 1fr) 5rem;
            align-items: center;
            border-bottom: 1px solid rgba(0, 0, 0, .15)
        }
        .row-grid > * {
            padding: .25rem .5rem;
            overflow-wrap: anywhere;
            min-width: 0
        }
        .row-grid.hdr {
            font-weight: 600;
            background: #f8f9fa
        }
        .p { background: #d1e7dd }
        .f { background: #f8d7da }
        .s { background: #e2e3e5 }
        details.case > summary {
            list-style: none;
            cursor: pointer
        }
        details.case > summary::-webkit-details-marker { display: none }
        details.case > summary > :nth-child(2)::before {
            content: "\\25B8  ";
            color: #6c757d
        }
        details.case[open] > summary > :nth-child(2)::before {
            content: "\\25BE  ";
            color: #6c757d
        }
        details.case:target, .row-grid:target { outline: 2px solid #0d6efd }
        .cbody {
            padding: .5rem 1rem;
            background: #fff
        }
        .kv {
            font-size: .85rem;
            margin-bottom: .5rem
        }
        pre {
            white-space: pre-wrap;
            word-break: break-word;
            background: #f8f9fa;
            padding: .5rem;
            border-radius: .25rem;
            margin-bottom: .5rem
        }
        section { margin-bottom: 1.5rem }
        section:target { background: #fff3cd }
        section.suite {
            border-left: 3px solid #dee2e6;
            padding-left: 1rem;
        }
        summary { cursor: pointer }
    """

    out.write(f"""
        <!doctype html>
        <html lang="en">
            <head>
                <meta charset="utf-8">
                <meta name="viewport" content="width=device-width, initial-scale=1">
                <title>{esc(title)}</title>
                <link rel="stylesheet" href="{BOOTSTRAP_CSS_URL}">
                <style>{extra_css}
                </style>
            </head>
            <body>
                <div class="container-fluid">
                    <h1>{esc(title)}</h1>
    """)


def write_doc_end(out):
    out.write("""
                </div>
                <script>
                    document.addEventListener('click', function(ev) {
                        var a = ev.target.closest('a.anchor');
                        if (a) {
                            ev.preventDefault();
                            location.hash=a.getAttribute('href');
                        }
                    });
                </script>
            </body>
        </html>
    """)


def write_summary(out, counts):
    out.write(f"""
        <div class="card mb-3">
            <div class="card-body">
                <h5 class="card-title">Summary</h5>
    """)
    write_counts(out, counts)
    out.write("""
            </div>
        </div>
    """)


def write_counts(out, counts):
    out.write(badge(counts[Ctr.TESTS], 'text-bg-dark', 'tests'))
    out.write(badge(counts[Ctr.PASSED], 'text-bg-success', 'passed'))
    out.write(badge(counts[Ctr.FAILURES], 'text-bg-danger', 'failed'))
    out.write(badge(counts[Ctr.ERRORS], 'text-bg-warning', 'errored'))
    out.write(badge(counts[Ctr.SKIPPED], 'text-bg-secondary', 'skipped'))


def write_controls(out):
    out.write("""
        <div class="mb-3">
            <button class="btn btn-sm btn-outline-secondary me-2"
                onclick="document.querySelectorAll('details.io').forEach(function(d){d.open=true;})">
                Expand all
            </button>
            <button class="btn btn-sm btn-outline-secondary me-2"
                onclick="document.querySelectorAll('details.io').forEach(function(d){d.open=false;})">
                Collapse all
            </button>
        </div>
    """)


def badge(count, cls, label):
    cls = cls if count > 0 else "text-bg-light"
    return f'<span class="badge me-1 {cls}">{count} {esc(label)}</span>'


def esc(t):
    return html.escape(str(t), quote=False)


def pass2_render_file(out, file, suites, suite_counter):
    suite_idx = 0

    for event, elem in ET.iterparse(file, events=('start', 'end')):
        tag = elem.tag
        if event == 'start':
            if tag == 'testsuite':
                suite = suites[suite_idx]
                suite_idx += 1
                ensure(suite.name == elem.get('name', ''), 'error: could not match the suite')
                write_suite_start(out, suite, file, suite_counter.value)
                suite_counter.inc()
            pass
        else:  # event == 'end'
            if tag == 'testsuite':
                write_suite_end(out)
            pass

    ensure(suite_idx == len(suites), 'error: not all suites were iterated')


def write_suite_start(out, suite, file, suite_counter):
    out.write(f"""
        <section id="s{suite_counter}" class="suite">
            <h2 class="sh">{esc(suite.name)}<a class="anchor" href="#s{suite_counter}">#</a></h2>
            <div class="mb-2">
    """)
    write_counts(out, suite.counts)
    out.write(f"""
                <small class="text-muted">{esc(suite.time) + 's' if suite.time else ''}</small>
            </div>
    """)

    meta = sorted(suite.props + [('xml', file)])
    out.write("""
            <details class="io mb-2"><summary>Meta</summary>
                <table class="table table-sm table-borderless w-auto kv"><tbody>
    """)
    for k, v in meta:
        out.write(f'<tr><td>{esc(k)}</td><td>{esc(v)}</td></tr>')
    out.write("""
                </tbody></table>
            </details>
    """)

    if suite.sysout:
        out.write(f"""
            <details class="io mb-2"><summary>system-out</summary>
                <pre>{esc(suite.sysout)}</pre>
            </details>
        """)

    if suite.syserr:
        out.write(f"""
            <details class="io mb-2"><summary>system-err</summary>
                <pre>{esc(suite.syserr)}</pre>
            </details>
        """)


def write_suite_end(out):
    out.write('</section>')


class Counter:
    def __init__(self, value=0):
        self.value = value

    def inc(self):
        self.value += 1


if __name__ == '__main__':
    sys.exit(main())
