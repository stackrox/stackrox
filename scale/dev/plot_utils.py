#!/usr/bin/env python3
"""
Shared utilities for performance test plotting scripts.
"""

import os

import numpy as np
import matplotlib.pyplot as plt


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


def add_trendline(x_data, y_data, label, color, linestyle='--',
                  max_iterations=10, sigma=2.0):
    """
    Add a linear trend line to the current plot and return the equation.

    Outliers are removed iteratively: after each linear fit, points whose
    residual exceeds `sigma` standard deviations of the residuals are dropped
    and the line is refit. This repeats until no outliers remain or
    `max_iterations` is reached (always keeping at least 2 points).

    Args:
        x_data: X-axis values
        y_data: Y-axis values
        label: Label for the trend line
        color: Color for the trend line
        linestyle: Line style for the trend line
        max_iterations: Maximum number of outlier-removal iterations
        sigma: Residual threshold (in standard deviations) for an outlier

    Returns:
        Tuple of (slope, intercept) or None if insufficient data
    """
    # Filter out None values
    valid_points = [(x, y) for x, y in zip(x_data, y_data) if y is not None]
    if len(valid_points) < 2:
        return None  # Need at least 2 points for a trend line

    x_valid = np.array([p[0] for p in valid_points], dtype=float)
    y_valid = np.array([p[1] for p in valid_points], dtype=float)

    # Fit linear trend line, iteratively removing residual outliers
    coeffs = np.polyfit(x_valid, y_valid, 1)
    for _ in range(max_iterations):
        residuals = y_valid - np.polyval(coeffs, x_valid)
        std = residuals.std()
        if std == 0:
            break  # perfect fit, nothing to remove
        keep = np.abs(residuals) <= sigma * std
        if keep.all() or keep.sum() < 2:
            break  # no outliers, or removing more would leave < 2 points
        x_valid = x_valid[keep]
        y_valid = y_valid[keep]
        coeffs = np.polyfit(x_valid, y_valid, 1)

    trend_y = np.polyval(coeffs, x_valid)

    # Plot trend line
    plt.plot(x_valid, trend_y, linestyle=linestyle, linewidth=1.5, color=color,
             alpha=0.7, label=label)

    # Return slope and intercept
    return (coeffs[0], coeffs[1])


def add_equation_text(equations, y_position=0.95):
    """
    Add trend line equations as text on the plot.

    Args:
        equations: List of tuples (label, slope, intercept, color)
        y_position: Vertical position for the text box (0-1, in axes coordinates)
    """
    if not equations:
        return

    equation_text = []
    for label, slope, intercept, color in equations:
        if slope is not None and intercept is not None:
            # Format equation: y = mx + b
            sign = '+' if intercept >= 0 else '-'
            equation_text.append(f"{label}: y = {slope:.4e}x {sign} {abs(intercept):.4f}")

    if equation_text:
        text_str = '\n'.join(equation_text)
        plt.text(0.02, y_position, text_str, transform=plt.gca().transAxes,
                fontsize=9, verticalalignment='top',
                bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))
