# Component-Specific Baseline Detection

## Overview

The plotting scripts now support component-specific baseline timestamps (t=0) to account for different component startup times.

## Baseline Logic

### Sensor
- **t=0 = earliest timestamp across all sensor metric files (CPU and memory)**
- Rationale: Sensor can start measuring immediately when it comes online
- Note: Uses minimum timestamp to ensure both CPU and memory metrics start at t=0

### Central & Central-DB
- **t=0 = when deployments >= 100**
- If deployments never reaches 100: use **90% of maximum deployment count**
- Rationale: These components need the cluster to be in a stable state before meaningful measurements

## Updated Scripts

### 1. plot-file-activity.py
Comparison plots between two test runs (e.g., with/without policy).

**New Usage:**
```bash
python3 plot-file-activity.py \
    results1/metrics_central_cpu.txt "Without Policy" \
    results2/metrics_central_cpu.txt "With Policy" \
    "Central CPU Usage" "CPU Cores" \
    results1 results2 central \
    output.png
```

**Parameters:**
- `results_dir1`, `results_dir2`: Result directories (for baseline detection)
- `component`: One of `sensor`, `central`, or `central-db`

**Legacy mode** (explicit base times) still works for backward compatibility.

### 2. plot-batch-comparison.py
Cross-batch analysis showing how metrics scale with event rate.

**No changes to usage** - baseline detection is automatic:
```bash
python3 plot-batch-comparison.py perf perf/comparison_plots
```

The script automatically:
- Detects component type from file paths
- Determines appropriate baseline for each component
- Uses component-specific baselines when calculating averages/maximums

### 3. plot-single-metric.py
Single metric time-series plots.

**New Usage:**
```bash
python3 plot-single-metric.py \
    results/metrics_central_cpu.txt \
    "Central CPU" "CPU Cores" \
    output.png \
    results central
```

**Parameters:**
- `results_dir`: Result directory (for baseline detection)
- `component`: One of `sensor`, `central`, or `central-db`

### 4. MakePlots-file-activity.sh
Updated to pass component information to plotting scripts.

**No changes to usage:**
```bash
./MakePlots-file-activity.sh without_policy_dir with_policy_dir output_dir
```

## Testing

Test baseline detection on a results directory:
```bash
python3 test-baseline-detection.py <results_dir>
```

Example output:
```
sensor:
  Baseline timestamp: 1754522400000ms
  Human-readable: 2025-08-06 16:20:00

central:
  Found deployments >= 100: 2557.0 at timestamp 1754522520000
  Baseline timestamp: 1754522520000ms
  Human-readable: 2025-08-06 16:22:00
```

## Implementation Details

### determine_baseline_timestamp(results_dir, component)
Core function that implements the baseline logic:
1. For sensor: reads both `metrics_sensor_cpu.txt` and `metrics_sensor_mem.txt`
   - Returns the **minimum** (earliest) timestamp across both files
   - This ensures both CPU and memory plots start at t=0
   - Handles cases where memory data starts before CPU data
2. For central/central-db: reads `metrics_deployments.txt`
   - Returns first timestamp where deployments >= 100
   - If never reaches 100, returns first timestamp where deployments >= 90% of max
   - Fallback: returns first timestamp if all else fails

### Changes to read functions
- `read_metric_average()` and `read_metric_max()` now accept optional `base_time` parameter
- Time windows (START_OFFSET, END_OFFSET) are relative to component-specific baseline
- This ensures fair comparison when components start at different times

## Benefits

1. **More accurate comparisons**: Components are aligned to their actual start of work, not arbitrary wall-clock time
2. **Handles variable startup**: Different runs may have different startup characteristics
3. **Automatic detection**: No manual timestamp calculation needed
4. **Backward compatible**: Legacy scripts using explicit base times still work

## Migration Notes

- Existing scripts using `MakePlots-file-activity.sh` automatically get the new behavior
- Scripts calling `plot-*.py` directly can use new component-based mode or continue with legacy explicit base times
- No changes needed to result directory structure or metric file formats
