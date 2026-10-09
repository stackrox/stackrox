#!/usr/bin/env python3
"""Validate ci/test-domains.toml for quality issues."""

import sys
import argparse
from pathlib import Path
from resolver import load_mapping
from decisionlib import parse_defaults

def validate_job_coverage(mapping):
    """Warn about jobs never explicitly mentioned in rules."""
    jobs_set = set(mapping.jobs)
    mentioned = set()

    for rule in mapping.rules:
        if rule.run_all or rule.skip_all or rule.default_all:
            continue  # Covers all jobs
        mentioned.update(rule.run)
        mentioned.update(rule.skip)
        mentioned.update(rule.default)

    never_mentioned = jobs_set - mentioned
    if never_mentioned:
        print(f"WARNING: Jobs only covered by 'remaining' rule: {sorted(never_mentioned)}")
        print("  Consider adding explicit rules for better clarity")

    return len(never_mentioned)

def validate_defaults_completeness(mapping, defaults_file):
    """Verify all jobs have defaults and vice versa."""
    with open(defaults_file, encoding='utf-8') as f:
        defaults = parse_defaults(f.read())

    jobs_set = set(mapping.jobs)
    defaults_set = set(defaults.keys())

    missing_defaults = jobs_set - defaults_set
    extra_defaults = defaults_set - jobs_set

    errors = 0
    if missing_defaults:
        print(f"ERROR: Jobs missing from decision-defaults: {sorted(missing_defaults)}")
        errors += 1
    if extra_defaults:
        print(f"ERROR: Unknown jobs in decision-defaults: {sorted(extra_defaults)}")
        errors += 1

    return errors

def main():
    parser = argparse.ArgumentParser(description='Validate test-domains.toml')
    parser.add_argument('--rules', default='ci/test-domains.toml', help='Rules file')
    parser.add_argument('--defaults', default='ci/decision-defaults', help='Defaults file')
    args = parser.parse_args()

    try:
        mapping = load_mapping(args.rules)
    except Exception as e:
        print(f"ERROR: Failed to load rules: {e}")
        return 1

    errors = 0
    warnings = 0

    # Check defaults completeness
    errors += validate_defaults_completeness(mapping, args.defaults)

    # Check job coverage (warnings only)
    warnings += validate_job_coverage(mapping)

    if errors > 0:
        print(f"\nValidation failed: {errors} error(s), {warnings} warning(s)")
        return 1
    elif warnings > 0:
        print(f"\nValidation passed with {warnings} warning(s)")
        return 0
    else:
        print("\nValidation passed")
        return 0

if __name__ == '__main__':
    sys.exit(main())
