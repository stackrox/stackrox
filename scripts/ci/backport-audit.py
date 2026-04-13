#!/usr/bin/env python3
"""
Backport PR audit tool - entry point.

This script validates backport PRs and Jira issues for release management.
"""

import sys
from backport_audit.__main__ import main

if __name__ == "__main__":
    sys.exit(main())
