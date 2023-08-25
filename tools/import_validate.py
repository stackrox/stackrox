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

def complete(data):
    blanks = data.get('blanks', [])
    for n in blanks[1:]:
        print("%s:%d: Too many blank lines in imports" % (data["file"], n))

PREIMPORT, IMPORTS, POSTIMPORT, EXTRACOMMENT, BLANKIMPORT = range(5)

TRANSITIONS = [
    (PREIMPORT, 'import (', IMPORTS),
    (PREIMPORT, '', PREIMPORT),
    (IMPORTS, '\t//', IMPORTS, comment),
    (IMPORTS, '\t_ ', IMPORTS, blank_import),
    (IMPORTS, '\t', IMPORTS),
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
