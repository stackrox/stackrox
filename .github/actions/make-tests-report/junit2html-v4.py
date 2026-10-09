#!/usr/bin/env python3

import argparse
import sys
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from enum import Enum
from pathlib import Path


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
        ok, suites = parse_file_pass1(f)
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

    print(grand_totals)

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


def parse_file_pass1(file):
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


if __name__ == '__main__':
    sys.exit(main())
