#!/usr/bin/env python3
"""
Thin YAML reformatter that normalizes Go-generated YAML to match PyYAML output.
This script has NO knowledge of CSV/bundle structure - it only normalizes formatting.

This is the "escape hatch" mentioned in the migration plan (Section 2.2).
All business logic remains in Go; this only handles YAML formatting quirks.
"""
import sys
import yaml

# Read YAML from stdin
doc = yaml.safe_load(sys.stdin)

# Write YAML to stdout with PyYAML's formatting
# Note: yaml.safe_dump() adds a trailing newline, and print() adds another
print(yaml.safe_dump(doc))
