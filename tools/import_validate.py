#!/usr/bin/env python3

import os
import sys
import re

def append(data, key, value):
    cur = data.get(key, [])
    cur.append(value)
    data[key] = cur

def add_blank(data):
    append(data, "blanks", data["num"])

def comment(data):
    append(data, "comment", data["num"])

def blank_import(data):
    '''Remove extra blank lines associated with _ imports'''
    num = data['num']
    if not 'comment' in data:
        return
    comments = data["comment"]
    del data['comment']
    while comments:
        num -= 1
        prev = comments.pop(-1)
        if prev < num:
            break
    # at this point, num == line of first comment
    blanks = data.get("blanks", [])
    if len(blanks) < 1:
        return
    if blanks[-1] == num - 1:
        blanks.pop(-1)

def check_imports(data):
    # We only tolerate blank lines between first and third party imports, as well as blank (_) imports
    s_line = data.get('line', '').lstrip()
    import_state = data.get('import_state', TPIMPORTS)
    # ignore . imports, but recognize 3rd party imports of the form github.com/user/proj;
    # dot imports are stripped of their leading dot.
    if s_line.startswith('.'):
        s_line = s_line[1:]

    if '.' in s_line and import_state == FPIMPORTS:
        data["import_state"] = TPIMPORTS
        check_blanks(data)

def check_blanks(data):
    did_run = data.get('check_blanks', False)
    blanks = data.get('blanks', [])
    import_state = data.get('import_state', FPIMPORTS)

    if import_state == FPIMPORTS or did_run:
        return # nothing to do here
    # If we're in 3rd party imports, we should only ever have one blank separating 1st and 3rd party imports
    # keep the last blank, as it's likely the correct one, as we're in the transition to 3rd party imports here
    data['blanks'] = blanks[:-1]
    data['check_blanks'] = True

def complete(data):
    blanks = data.get('blanks', [])
    for n in blanks:
        print("%s:%d: Too many blank lines in imports" % (data["file"], n))

PREIMPORT, IMPORTS, POSTIMPORT, EXTRACOMMENT, BLANKIMPORT = range(5)
FPIMPORTS, TPIMPORTS = 1, 3 # We differentiate between first party (FPIMPORTS) and third party (TPIMPORTS) imports

TRANSITIONS = [
    (PREIMPORT, 'import (', IMPORTS),
    (PREIMPORT, '', PREIMPORT),
    (IMPORTS, '\t//', IMPORTS, comment),
    (IMPORTS, '\t_ ', IMPORTS, blank_import),
    (IMPORTS, '\t', IMPORTS, check_imports),
    (IMPORTS, '\n', IMPORTS, add_blank),
    (IMPORTS, ')', POSTIMPORT, complete),
    (POSTIMPORT, '', POSTIMPORT),
]

def main():
    ok = True
    for gofile in sys.argv[1:]:
        fileok = scan(gofile)
        ok = ok and fileok
    if not ok:
        sys.exit(1)

def scan(gofile):
    data = {"file": gofile}
    state = PREIMPORT
    for num, line in enumerate(open(gofile), start = 1):
        data["num"] = num
        data["line"] = line
        data["state"] = state
        data["import_state"] = FPIMPORTS # Start with first party imports
        matched = False
        for t in TRANSITIONS:
            if state == t[0] and line.startswith(t[1]):
                matched = True
                data["newstate"] = t[2]
                state = t[2]
                if len(t) == 4:
                    t[3](data)
                break
        if not matched:
            print("Unmatched state: %s, line %s" % (state, repr(line)))
    return len(data.get("blanks", [])) <= 1

if __name__ == '__main__':
    main()
