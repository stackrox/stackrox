#!/usr/bin/env python3
"""
Shared utilities for performance test plotting scripts.
"""

import os


def determine_baseline_timestamp(results_dir, component):
    """
    Determine the baseline timestamp (t=0) for a given component.

    Args:
        results_dir: Directory containing metrics files
        component: Component name (e.g., 'sensor', 'central', 'central-db')

    Returns:
        Baseline timestamp in milliseconds, or None if cannot be determined

    Logic:
        - For sensor: use earliest timestamp across all sensor metric files
        - For central/central-db: use timestamp when deployments >= 100
          (or 90% of max if never reaches 100)
    """
    if component == 'sensor':
        # For sensor, use earliest timestamp across all sensor metric files
        sensor_files = [
            os.path.join(results_dir, 'metrics_sensor_cpu.txt'),
            os.path.join(results_dir, 'metrics_sensor_mem.txt')
        ]

        min_timestamp = None
        for sensor_file in sensor_files:
            if not os.path.exists(sensor_file):
                continue

            with open(sensor_file, 'r') as f:
                for line in f:
                    parts = line.strip().split()
                    if len(parts) == 2:
                        try:
                            ts = int(parts[0])
                            if min_timestamp is None or ts < min_timestamp:
                                min_timestamp = ts
                            break  # Only need first timestamp from each file
                        except ValueError:
                            continue

        return min_timestamp

    elif component in ['central', 'central-db']:
        # For central/central-db, use timestamp when deployments >= 100
        deployments_file = os.path.join(results_dir, 'metrics_deployments.txt')
        if not os.path.exists(deployments_file):
            return None

        timestamps = []
        values = []

        with open(deployments_file, 'r') as f:
            for line in f:
                parts = line.strip().split()
                if len(parts) == 2:
                    try:
                        ts, val = int(parts[0]), float(parts[1])
                        timestamps.append(ts)
                        values.append(val)
                    except ValueError:
                        continue

        if not timestamps:
            return None

        # Find first timestamp where deployments >= 100
        for ts, val in zip(timestamps, values):
            if val >= 100:
                return ts

        # If never reaches 100, use 90% of maximum
        max_deployments = max(values)
        threshold = 0.9 * max_deployments
        for ts, val in zip(timestamps, values):
            if val >= threshold:
                return ts

        # Fallback: use first timestamp
        return timestamps[0]

    return None
