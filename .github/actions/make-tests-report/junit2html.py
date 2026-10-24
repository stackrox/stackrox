#!/usr/bin/env python3

"""
Converts JUnit XML files in HTML single-file report.
"""

import argparse
import datetime
import html
import sys
import tempfile
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from enum import Enum
from pathlib import Path
from urllib.parse import urlparse

BOOTSTRAP_CSS_URL = "https://cdn.jsdelivr.net/npm/bootstrap@5.3.8/dist/css/bootstrap.min.css"


def main():
    args = parse_args()
    args.meta.append(['generated-on', datetime.datetime.now(datetime.timezone.utc).isoformat()])

    input_dir = Path(args.input_dir)
    input_dir.stat()  # Raise error if input_dir does not exist.

    xml_files = sorted(input_dir.rglob('*.xml'), key=lambda p: str(p))
    print(f'Found XML files: {len(xml_files)}')

    grand_totals = make_counts()
    valid_files = []

    # The script uses XML streaming API in order not to load the entire documents in memory. Therefore, the processing
    # is done in two passes. On the first pass, invalid files are filtered out, and for valid ones per-suite
    # information is loaded as well as test counts are aggregated. On the second pass, HTML report is rendered from
    # the aggregated info and contents of the valid files.

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
        print(f'found {file_totals[Kind.TESTS]} tests')
        valid_files.append((f, suites))

    if not valid_files:
        raise RuntimeError('No valid JUnit reports found')

    output_file = Path(args.output_file)

    with tempfile.NamedTemporaryFile(mode='wt', encoding='utf-8', dir=output_file.parent, delete=False,
                                     prefix=output_file.name + '-tmp-') as tmp_out:
        tmp_name = tmp_out.name
        print('Rendering HTML...')
        pass2_render(tmp_out, args.title, grand_totals, valid_files, args.meta)

    Path(tmp_name).rename(output_file)

    size_mib = output_file.stat().st_size / 1024.0 / 1024
    print(f'Wrote {output_file} ({size_mib:.2f} MiB, {len(valid_files)} input files, ' +
          f'{grand_totals[Kind.TESTS]} tests, {grand_totals[Kind.FAILURES]} failed tests, ' +
          f'{grand_totals[Kind.ERRORS]} test errors, {grand_totals[Kind.SKIPPED]} skipped tests)')

    return 0


def parse_args():
    parser = argparse.ArgumentParser(add_help=True,
                                     description='Convert JUnit XML files in HTML single-file report.')
    parser.add_argument('input_dir', metavar='input-dir', help='Input directory with JUnit XML files.')
    parser.add_argument('output_file', metavar='output-file', help='Output HTML report file.')
    parser.add_argument('--title', default='Test Report', help='Title to put in the report file.')
    parser.add_argument('--meta', action='append', default=[], metavar='KEY=VALUE',
                        help='Additional metadata to include in the report. Can be repeated multiple times. '
                             'Values for keys ending with _URL are rendered as links.')
    result = parser.parse_args(sys.argv[1:])

    parsed_meta = []
    for item in result.meta:
        parts = item.split('=', maxsplit=1)
        if len(parts) != 2:
            raise ValueError(f'--meta argument in wrong format. Expected "KEY=VALUE", got "{item}"')
        parsed_meta.append(parts)
    result.meta = parsed_meta

    return result


class Kind(Enum):
    """
    For counting tests, displaying badges, and selecting colors via CSS classes.
    """
    TESTS = {
        'id': 1,
        'badge-color-class': 'text-bg-dark',
        'summary-badge-label': 'tests',
    }
    PASSED = {
        'id': 2,
        'badge-color-class': 'text-bg-success',
        'summary-badge-label': 'passed',
        'case-row-class': 'p',
        'case-badge-label': 'PASSED',
    }
    FAILURES = {
        'id': 3,
        'badge-color-class': 'text-bg-danger',
        'summary-badge-label': 'failed',
        'case-row-class': 'f',
        'case-badge-label': 'TEST FAILED',
    }
    ERRORS = {
        'id': 4,
        'badge-color-class': 'text-bg-warning',
        'summary-badge-label': 'errored',
        'case-row-class': 'f',
        'case-badge-label': 'TEST ERROR',
    }
    SKIPPED = {
        'id': 5,
        'badge-color-class': 'text-bg-secondary',
        'summary-badge-label': 'skipped',
        'case-row-class': 's',
        'case-badge-label': 'TEST SKIPPED',
    }

    @property
    def badge_color_class(self):
        return self.value['badge-color-class']

    @property
    def summary_badge_label(self):
        return self.value['summary-badge-label']

    @property
    def case_row_class(self):
        return self.value['case-row-class']

    @property
    def case_badge_label(self):
        return self.value['case-badge-label']


def make_counts():
    return {
        Kind.TESTS: 0,
        Kind.PASSED: 0,
        Kind.FAILURES: 0,
        Kind.ERRORS: 0,
        Kind.SKIPPED: 0,
    }


def sum_up(acc, item):
    for k in acc.keys():
        acc[k] += item[k]


@dataclass
class TestSuite:
    name: str
    counts: dict[Kind, int]
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
                    assert test_suite is None, 'nested testsuite-s are not supported'
                    test_suite = make_test_suite(elem)
                    suites.append(test_suite)
                elif tag == 'testcase':
                    assert test_suite is not None, 'error: testcase outside testsuite'
                    assert test_case is None, 'nested testcase-s are not supported'
                    test_case = elem
            else:  # event == 'end'
                if tag == 'testsuite':
                    assert (test_suite.counts[Kind.TESTS] * 2 ==
                            sum(test_suite.counts.values())), 'something is wrong with our tests counting'
                    test_suite = None
                elif tag == 'testcase':
                    test_case = None
                    test_suite.counts[Kind.TESTS] += 1
                    status, _ = classify_case(elem)
                    test_suite.counts[status] += 1
                elif tag == 'system-out':
                    if test_case is None:
                        assert not test_suite.sysout, 'multiple system-out elements are not supported'
                        test_suite.sysout = text_of(elem)
                elif tag == 'system-err':
                    if test_case is None:
                        assert not test_suite.syserr, 'multiple system-err elements are not supported'
                        test_suite.syserr = text_of(elem)
                elif tag == 'properties':
                    if test_case is None:
                        test_suite.props.extend(extract_props(elem))
        return True, suites
    except ET.ParseError:
        return False, []


def make_test_suite(el):
    return TestSuite(
        name=el.get('name', ''),
        counts=make_counts(),
        time=el.get('time', ''),
        props=collect_attributes(el, except_for={'name', 'time'}),
        sysout='',
        syserr='',
    )


def collect_attributes(el, except_for):
    props = []
    for attr_name, attr_value in el.items():
        if attr_name in except_for:
            continue
        props.append((attr_name, attr_value))
    return props


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


def classify_case(testcase_el):
    mapping = [('failure', Kind.FAILURES), ('error', Kind.ERRORS), ('skipped', Kind.SKIPPED)]
    for el_name, status in mapping:
        outcome_el = testcase_el.find(el_name)
        if outcome_el is not None:
            return status, outcome_el
    assert not testcase_el.get('status'), 'status override attribute is not supported'
    return Kind.PASSED, None


def pass2_render(out, title, grand_totals, valid_files, metadata):
    write_doc_start(out, title)
    write_summary(out, grand_totals, metadata)
    write_controls(out)

    href_suite_counter = Counter()
    href_case_counter = Counter()
    for file, suites in valid_files:
        pass2_render_file(out, file, suites, href_suite_counter, href_case_counter)

    write_doc_end(out)


class Counter:
    """
    Simply encapsulates a value that can be incremented across multiple function calls.
    """

    def __init__(self, value=0):
        self.value = value

    def inc(self):
        self.value += 1


def write_doc_start(out, title):
    extra_css = """
        body { padding: 1rem }
        h1 { margin-bottom: 1rem }
        .anchor {
            visibility: hidden;
            margin-left: .4rem;
            text-decoration: none;
            color: #6c757d;
            font-weight: normal;
        }
        .sh:hover .anchor, .row-grid:hover .anchor { visibility: visible }
        .row-grid {
            display: grid;
            grid-template-columns:9rem minmax(10rem, 1fr) 5rem;
            align-items: center;
            border-bottom: 1px solid rgba(0, 0, 0, .15);
        }
        .row-grid > * {
            padding: .25rem .5rem;
            overflow-wrap: anywhere;
            min-width: 0;
        }
        .row-grid.hdr {
            font-weight: 600;
            background: #f8f9fa;
        }
        .p { background: #d1e7dd }
        .f { background: #f8d7da }
        .s { background: #e2e3e5 }
        details.case > summary {
            list-style: none;
            cursor: pointer;
        }
        details.case > summary::-webkit-details-marker { display: none }
        details.case > summary.d > :nth-child(2)::before {
            content: "\\25B8  ";
            color: #6c757d;
        }
        details.case[open] > summary.d > :nth-child(2)::before {
            content: "\\25BE  ";
            color: #6c757d;
        }
        details.case:target, .row-grid:target { outline: 2px solid #0d6efd }
        .dtl {
            padding: .5rem 1rem;
            background: #fff;
        }
        .kv {
            font-size: .85rem;
            margin-bottom: .5rem;
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


def esc(t):
    return html.escape(str(t), quote=False)


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


def write_summary(out, counts, metadata):
    out.write(f"""
        <div class="card mb-3">
            <div class="card-body">
                <h5 class="card-title">Summary</h5>
    """)

    write_counts(out, counts)

    out.write('<details class="io mb-2"><summary>Meta</summary>')
    write_key_value_table(out, metadata)
    out.write('</details>')

    out.write("""
            </div>
        </div>
    """)


def write_counts(out, counts):
    for status, count in counts.items():
        cls = status.badge_color_class if count > 0 else "text-bg-light"
        out.write(f'<span class="badge me-1 {cls}">{count} {esc(status.summary_badge_label)}</span>')


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


def pass2_render_file(out, file, suites, suite_counter, case_counter):
    suite_idx = 0
    suite = None
    header_written = False

    for event, elem in ET.iterparse(file, events=('start', 'end')):
        tag = elem.tag
        if event == 'start':
            if tag == 'testsuite':
                suite = suites[suite_idx]
                suite_idx += 1
                assert suite.name == elem.get('name', ''), 'error: could not match the suite'
                write_suite_start(out, suite, file, suite_counter.value)
                suite_counter.inc()
                header_written = False
        else:  # event == 'end'
            if tag == 'testsuite':
                write_suite_end(out)
            elif tag == 'testcase':
                if not header_written:
                    header_written = True
                    write_cases_header(out)
                write_case(out, elem, suite.name, case_counter.value)
                case_counter.inc()

    assert suite_idx == len(suites), 'error: not all suites were iterated'


def write_suite_start(out, suite, file, suite_index):
    out.write(f"""
        <section id="s{suite_index}" class="suite">
            <h2 class="sh">{esc(suite.name)}<a class="anchor" href="#s{suite_index}">#</a></h2>
            <div class="mb-2">
    """)
    write_counts(out, suite.counts)
    if suite.time:
        out.write(f'<small class="text-muted">{esc(suite.time)}s</small>')
    out.write("""
            </div>
    """)

    out.write('<details class="io mb-2"><summary>Meta</summary>')
    write_key_value_table(out, suite.props + [('xml', file)])
    out.write('</details>')

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


def write_key_value_table(out, props):
    props = sorted(props)
    out.write('<table class="table table-sm table-borderless w-auto kv"><tbody>')
    for k, v in props:
        value = esc(v)
        if k.lower().endswith('_url'):
            parsed = urlparse(v)
            if parsed.scheme and parsed.scheme in {'http', 'https'}:
                value = f'<a href="{esc(v)}" target="_blank">{esc(v)}</a>'
        out.write(f'<tr><td>{esc(k)}</td><td>{value}</td></tr>')
    out.write('</tbody></table>')
    pass


def write_cases_header(out):
    out.write('<div class="row-grid hdr"><div>Status</div><div>Test</div><div>Time (s)</div></div>')


def write_case(out, case_el, suite_name, case_index):
    time = case_el.get('time', '')

    display_name = get_case_display_name(
        suite_name=suite_name, name=case_el.get('name', ''), classname=case_el.get('classname', ''))

    props = collect_attributes(case_el, except_for={'time', 'name', 'classname'})
    # Will be easy to add properties to `props`, but I had no such data to test on.
    assert case_el.find('properties') is None, 'properties under testcase are not supported'

    status, outcome_el = classify_case(case_el)

    details = []
    if outcome_el is not None:
        # 'message' and 'type' attributes are only present for Groovy tests when these get skipped, and it looks so:
        # <skipped message="org.opentest4j.TestAbortedException: Ignored via @IgnoreIf" type="org.opentest4j.TestAbortedException">
        # Therefore, we can take only 'message' and have a fallback for unseen future cases.
        message = outcome_el.get('message', '') or outcome_el.get('type', '')
        if message:
            details.append(f'<div><strong>{esc(message)}</strong></div>')
        trace = text_of(outcome_el).strip()
        if trace:
            details.append(f'<pre>{esc(trace)}</pre>')

    sysout = ''.join(text_of(el) for el in case_el.findall('system-out'))
    if sysout:
        details.append(f'<div class="fw-semibold small">system-out</div><pre>{esc(sysout)}</pre>')
    syserr = ''.join(text_of(el) for el in case_el.findall('system-err'))
    if syserr:
        details.append(f'<div class="fw-semibold small">system-err</div><pre>{esc(syserr)}</pre>')

    out.write(f"""
            <details class="io case" id="t{case_index}">
                <summary class="{'d' if details or props else ''} row-grid {status.case_row_class}">
                    <div>
                        <span class="badge {status.badge_color_class}">{status.case_badge_label}</span>
                        <a class="anchor" href="#t{case_index}">#</a>
                    </div>
                    <div>{esc(display_name)}</div>
                    <div>{esc(time) or '&nbsp;'}</div>
                </summary>
    """)
    if details or props:
        out.write('<div class="dtl">')
        if props:
            write_key_value_table(out, props)
        for d in details:
            out.write(d)
        out.write('</div>')
    out.write(f"""
            </details>
    """)


def get_case_display_name(suite_name, name, classname):
    if not classname:
        return name
    elif classname == suite_name:
        # Converter for Go unit tests does this, we don't want duplication.
        # <testsuite name="github.com/stackrox/rox/central/administration/events/handler" ...
        #   <testcase name="TestEventsHandler" classname="github.com/stackrox/rox/central/administration/events/handler" ...
        return name
    elif name in classname:
        # UI tests tend to do this.
        # <testsuite name="Dashboard SummaryCounts" ...
        #   <testcase name="should display 6 counts for Analyst role" classname="Dashboard SummaryCounts should display 6 counts for Analyst role">
        return classname
    else:
        return f'{classname} / {name}'


if __name__ == '__main__':
    sys.exit(main())
