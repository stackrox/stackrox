# File Activity Performance Testing - Quick Reference

Quick commands for running file activity performance tests.

## Quick Start

```bash
# Single test
./perf-test-file-activity.sh 1 10m file-activity-batch-100 true

# Full test suite (all batch sizes, with and without policy)
./run-file-activity-tests.sh 1 10m

# Generate plots
./MakeAllPlots-file-activity.sh perf 1 10m
python3 plot-batch-comparison.py perf
```

## Available Workloads

| Workload | Batch Size | Event Rate | Use Case |
|----------|------------|------------|----------|
| file-activity-batch-10 | 10 | 100/sec | Baseline |
| file-activity-batch-50 | 50 | 500/sec | Low |
| file-activity-batch-100 | 100 | 1000/sec | Medium |
| file-activity-batch-250 | 250 | 2500/sec | High |
| file-activity-batch-500 | 500 | 5000/sec | Stress |

## Scripts Overview

### Test Execution
- `perf-test-file-activity.sh` - Run single test
- `run-file-activity-tests.sh` - Run all configurations

### Analysis
- `CheckDB-file-activity.sh` - Query alert counts
- `prometheus-query-file-activity.sh` - Collect metrics

### Visualization
- `plot-file-activity.py` - Basic comparison plot
- `MakePlots-file-activity.sh` - Generate all plots for one config
- `MakeAllPlots-file-activity.sh` - Generate plots for all configs
- `plot-batch-comparison.py` - Scaling analysis across batch sizes

### Policy
- `enable-file-activity-policy.sh` - Create test policy

## Results Location

```
perf/
├── file_activity_results_1_10m_file-activity-batch-10_policy_false/
├── file_activity_results_1_10m_file-activity-batch-10_policy_true/
├── file_activity_results_1_10m_file-activity-batch-50_policy_false/
├── file_activity_results_1_10m_file-activity-batch-50_policy_true/
├── ...
├── plots_file-activity-batch-10/
├── plots_file-activity-batch-50/
├── ...
└── comparison_plots/
    ├── central_cpu_vs_rate.png
    ├── central_mem_vs_rate.png
    ├── alerts_count_vs_rate.png
    └── policy_cpu_overhead.png
```

## Example: Complete Test Run

```bash
# 1. Run all tests (takes ~3-4 hours for full suite)
./run-file-activity-tests.sh 1 10m

# 2. Generate individual comparison plots
./MakeAllPlots-file-activity.sh perf 1 10m

# 3. Generate scaling analysis
python3 plot-batch-comparison.py perf

# 4. Review results
ls -lh perf/comparison_plots/
```

## Quick Test (5 minutes)

```bash
# Test just one configuration for validation
./perf-test-file-activity.sh 1 5m file-activity-batch-10 false

# Check results
ls -lh perf/file_activity_results_1_5m_file-activity-batch-10_policy_false/
```

## Full Documentation

See `FILE_ACTIVITY_TESTING.md` for comprehensive documentation.
