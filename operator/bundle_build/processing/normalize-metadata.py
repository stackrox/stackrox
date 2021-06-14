#!/usr/bin/env python3
"""
Sort consecutive lines in a file that all start from common prefix, provided as command-line argument.

Use this to "normalize" bundle Dockerfile and bundle metadata where strings can get randomly reordered upon
regeneration.
"""

import sys

if len(sys.argv) != 2:
    print(f'usage: {sys.argv[0]} <prefix>', file=sys.stderr)
    sys.exit(1)

prefix = sys.argv[1]

curr_block = []

for line in sys.stdin:
    line = line.rstrip('\r\n')
    if line.startswith(prefix):
        curr_block.append(line)
        continue
    if curr_block:
        curr_block.sort()
        print(*curr_block, sep='\n')
        curr_block = []
    print(line)

if curr_block:
    curr_block.sort()
    print(*curr_block, sep='\n')
