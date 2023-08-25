#!/usr/bin/env python3
"""
Script that fixes some lint issues automatically.
"""
from __future__ import print_function
from collections import defaultdict
import os
import re
import sys

# Define the states.
FILE_BEGINNING, SUMMARY_SEEN, IN_VIOLATIONS_FOR_FILE, BETWEEN_FILES, DONE = range(5)


def handle_indentation(filename, line_no, message, src, meta):

    match = re.match(r'.*: Expected column (\d+) but was (\d+)', message)
    assert match
    expected, actual = int(match.group(1)), int(match.group(2))

    full_filename = os.path.join(meta['src_dir'], filename)

    offset = meta['line_number_offsets'][filename]
    line_no = line_no + offset

    new_lines = []
    rewritten = []
    with open(full_filename, 'r') as f:
        i = 0
        for line in f:
            i += 1
            if i != line_no:
                new_lines.append(line)
                continue
            # Sanity check assertion
            assert line.strip().split() == src.split(), "{}:{} {} -> {}".format(
                full_filename, line_no, line.strip(), src)

            if line[:actual-1] == ' '*(actual-1) and line[actual-1] != ' ':
                rewritten = [' '*(expected-1) + line[actual-1:]]
            # This happens in cases like "and:", "when:", "cleanup:", etc
            # which codenarc doesn't seem to handle well.
            # The logic here may not make much sense, but that's only because
            # it matches the codenarc output, which itself does not make
            # much sense.
            elif line[:expected] == ' '*(expected) and line[expected] != ' ':
                assert line[actual-1] == '"'
                rewritten = [
                    line[1:actual-1].rstrip() + '\n',
                    (' '*(expected-1)) + line[actual-1:]
                ]
            assert len(rewritten) > 0
            new_lines.extend(rewritten)
    if len(rewritten) > 1:
        meta['line_number_offsets'][filename] += (len(rewritten) - 1)
    with open(full_filename, 'w') as f:
        f.write(''.join(new_lines))


def handle_rule(filename, line_no, rule, message, src, meta):
    # Currently handle only indentation.
    handler_func = {
        'Indentation': handle_indentation,
    }.get(rule)
    if handler_func:
        handler_func(filename, line_no, message, src, meta)


def handle_file_beginning(line, state, meta):
    if not line.startswith('Summary:'):
        return FILE_BEGINNING
    # Exit early if no violations
    for elem in line.split():
        if elem.startswith('FilesWithViolations'):
            num_files_with_violations = int(elem.split('=')[1])
            if num_files_with_violations == 0:
                print("No files with violations, you're good!")
                return DONE
            else:
                meta['num_violation_files'] = num_files_with_violations
    return SUMMARY_SEEN


def handle_summary_seen_or_in_between_files(line, state, meta):
    if line.startswith('[CodeNarc'):
        return DONE
    if line.startswith('File:'):
        meta['current_file'] = line.strip().split()[1]
        return IN_VIOLATIONS_FOR_FILE
    return state


def handle_in_violations_for_file(line, state, meta):
    line = line.strip()
    if len(line) == 0:
        meta['processed_files'].append(meta['current_file'])
        del meta['current_file']
        return BETWEEN_FILES
    assert line.startswith('Violation:'), line
    
    violation_info = {
        'Line': '',
        'Rule': '',
        'Msg': '',
        'Src': '',
    }

    def add_match(key, match_list):
        violation_info[key] = ' '.join(match_list).strip('[]')

    sp_line = line.split()
    curr_match = []
    for elem in sp_line:
        if len(curr_match) > 0:
            curr_match.append(elem)
            if elem.endswith(']'):
                add_match(curr_key, curr_match)
                curr_match = []
            continue
        if '=' in elem:
            before, after = elem.split('=')
            if '[' in after:
                curr_key = before
                curr_match = [after]
                if after.endswith(']'): # Special case
                    add_match(curr_key, curr_match)
                    curr_match = []
                continue
            violation_info[before] = after
    handle_rule(meta['current_file'], int(violation_info['Line']), violation_info['Rule'],
        violation_info['Msg'], violation_info['Src'], meta)
    return state


def handle_line(line, state, meta):
    return {
        FILE_BEGINNING: handle_file_beginning,
        SUMMARY_SEEN: handle_summary_seen_or_in_between_files,
        IN_VIOLATIONS_FOR_FILE: handle_in_violations_for_file,
        BETWEEN_FILES: handle_summary_seen_or_in_between_files,
    }[state](line, state, meta)


def handle_file(filename, src_dir):
    with open(filename, 'r') as f:
        curr_state = FILE_BEGINNING
        meta = {
            'src_dir': src_dir,
            'processed_files': [],
            'num_violation_files': 0,
            'line_number_offsets': defaultdict(int),
        }
        for line in f:
            curr_state = handle_line(line, curr_state, meta)
            if curr_state == DONE:
                break
        assert curr_state == DONE, "File ended in an unexpected state: {}".format(curr_state)
        assert len(meta['processed_files']) == meta['num_violation_files']

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: {} <path to qa_tests_backend>".format(sys.argv[0]))
        sys.exit(1)

    qa_tests_backend_path = sys.argv[1]
    report_dir = os.path.join(qa_tests_backend_path, 'build', 'reports', 'codenarc')
    for filename in os.listdir(report_dir):
        if not filename.endswith('.txt'):
            continue
        print("Processing {}...".format(filename))
        handle_file(
            os.path.join(report_dir, filename),
            # The source path will be src/{main, test}/groovy/
            os.path.join(qa_tests_backend_path, 'src', filename[:-4], 'groovy')
        )
        print()
