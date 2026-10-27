#!/usr/bin/env python3
"""
Test script to verify component-specific baseline detection.
"""

import sys
import os
from datetime import datetime
from plot_utils import determine_baseline_timestamp

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: test-baseline-detection.py <results_dir>")
        sys.exit(1)

    results_dir = sys.argv[1]

    print(f"Testing baseline detection for: {results_dir}")
    print()

    for component in ['sensor', 'central', 'central-db']:
        print(f"{component}:")
        baseline = determine_baseline_timestamp(results_dir, component)
        if baseline:
            print(f"  Baseline timestamp: {baseline}ms")
            dt = datetime.fromtimestamp(baseline / 1000.0)
            print(f"  Human-readable: {dt}")
        else:
            print(f"  Could not determine baseline")
        print()
