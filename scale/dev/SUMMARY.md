# File Activity Performance Testing - Setup Summary

## What Was Created

### 1. Main Test Scripts

- **perf-test-file-activity.sh** - Orchestrates a single test run
  - Deploys ACS with specified workload
  - Optionally enables file activity policy
  - Collects diagnostic bundles, DB metrics, and Prometheus metrics
  - Runs K6 load test

- **run-file-activity-tests.sh** - Batch runner for all configurations
  - Tests all 5 batch sizes (10, 50, 100, 250, 500)
  - Runs each with and without policy (10 total tests)
  - Automatically cleans up between tests

### 2. Supporting Scripts

- **CheckDB-file-activity.sh** - Queries alerts table for file activity policy violations
- **enable-file-activity-policy.sh** - Creates runtime policy matching fake file paths  
- **prometheus-query-file-activity.sh** - Collects CPU, memory, and table size metrics

### 3. Workload Configurations

Created 5 workload files in `scale/workloads/`:

| File | Batch Size | Event Rate | Use Case |
|------|------------|------------|----------|
| file-activity-batch-10.yaml | 10 | 100/sec | Baseline |
| file-activity-batch-50.yaml | 50 | 500/sec | Low |
| file-activity-batch-100.yaml | 100 | 1000/sec | Medium |
| file-activity-batch-250.yaml | 250 | 2500/sec | High |
| file-activity-batch-500.yaml | 500 | 5000/sec | Stress test |

All use **100ms interval** (easier to calculate rate than varying interval).

### 4. Documentation

- **FILE_ACTIVITY_TESTING.md** - Comprehensive guide covering:
  - How file activity differs from process events
  - Workload parameters
  - Test workflow
  - Results interpretation
  - Analysis questions

## Key Technical Decisions

### Policy Matching
The test policy (`enable-file-activity-policy.sh`) now correctly matches on **File Path** field with patterns that correspond to the directories used in fake file activity generation:
- `/etc/security/*`
- `/etc/pam.d/*`
- `/etc/ssh/*`
- `/var/log/*`
- `/var/run/*`
- `/tmp/*`
- `/etc/kubernetes/*`
- `/etc/cni/*`
- `/etc/sysconfig/*`
- `/etc/audit/*`

These match the `fileActivityDirs` array in `sensor/kubernetes/fake/fileactivity.go`.

### Database Tables
Focus on:
- **alerts** - Primary table for policy violations
- **deployments** - File activity context

Removed process-related tables (process_indicators, process_baselines, process_baseline_results) as they're not relevant for file activity.

### Event Processing Flow
```
Fake Workload
  ↓
FakeFileActivityEvent
  ↓
File System Pipeline (handleFakeFileActivityEvent)
  ↓
buildIndicator + translateWithIndicator  
  ↓
detector.ProcessFileAccess
  ↓
Policy Evaluation
  ↓
Alerts (if policy matches)
```

## Important Differences from Process Baseline Tests

| Aspect | Process Baselines | File Activity |
|--------|------------------|---------------|
| **Event Generation** | Per-pod (scales with deployments) | Global fixed rate |
| **Rate Formula** | pods × (1/processInterval) | batchSize / activityInterval |
| **Primary Table** | process_baselines | alerts |
| **Policy Type** | Process execution detection | File access detection |
| **Event Source** | DEPLOYMENT_EVENT (process) | DEPLOYMENT_EVENT (file access) |
| **Matching Field** | Process Name | File Path |

## How to Run

### Single Test
```bash
cd scale/dev
./perf-test-file-activity.sh 1 10m file-activity-batch-100 true
```

### Full Suite
```bash
cd scale/dev
./run-file-activity-tests.sh 1 10m
```

This runs 10 tests (5 batch sizes × 2 policy states), taking approximately 3-4 hours total.

### Quick Start
For initial validation, test just one configuration:
```bash
./perf-test-file-activity.sh 1 5m file-activity-batch-10 false
```

## Results Location

Results stored in:
```
perf/file_activity_results_<sensors>_<runtime>_<workload>_policy_<true|false>/
```

Each directory contains:
- `diagnostic_bundle_1/` and `diagnostic_bundle_2/`
- `file_activity_alerts_1.txt` and `file_activity_alerts_2.txt` (if policy enabled)
- `metrics/*` - Time series data for CPU, memory, table sizes
- `k6_load_test.txt` - K6 test output

## Analysis Workflow

1. **Baseline Performance** (no policy)
   ```bash
   grep "total_file_activity_alerts" perf/file_activity_results_*/file_activity_alerts_*.txt
   # Should be 0 or very low
   ```

2. **With Policy Performance**
   ```bash
   # Check alert generation rate
   grep "count" perf/file_activity_results_*_policy_true/file_activity_alerts_2.txt
   ```

3. **Resource Usage**
   ```bash
   # Plot CPU/memory from metrics directory
   # Compare policy_true vs policy_false for same batch size
   ```

4. **Table Growth**
   ```bash
   # Check alerts table size growth
   tail perf/file_activity_results_*/metrics/*_alerts_bytes.txt
   ```

## Next Steps

1. **Validate** - Run single test with batch-10 to verify everything works
2. **Adjust** - If needed, modify policy in `enable-file-activity-policy.sh`
3. **Baseline** - Run full suite without policy to establish baseline
4. **Policy Test** - Run full suite with policy to measure impact
5. **Analyze** - Compare results, create plots, document findings

## Potential Issues & Solutions

### Issue: Policy doesn't create alerts
- **Check**: Is feature flag enabled? `ROX_SENSITIVE_FILE_ACTIVITY=true`
- **Check**: Does policy JSON parse correctly?
- **Debug**: Check Central logs for policy errors

### Issue: Very high alert count
- **Expected**: Policy matches ALL file activity in those directories
- **This is OK**: We want to stress test the system
- **Monitor**: Watch for alert processing lag

### Issue: Sensor runs out of memory
- **Solution**: Reduce batch size or increase sensor memory limit
- **Check**: `kubectl top pod -n stackrox`

### Issue: Database grows very large
- **Expected**: High event rates create many alerts
- **Monitor**: Check disk space on PVC
- **Solution**: Reduce test duration or batch size

## Verification Checklist

Before running full test suite:

- [ ] Fake file activity branch is active
- [ ] Workload configs exist in `scale/workloads/`
- [ ] All scripts are executable (`chmod +x`)
- [ ] Cluster has sufficient resources
- [ ] Storage PVC has enough space
- [ ] Feature flag enabled (if needed)
- [ ] Ran single test successfully

## Questions for Analysis

1. **Performance**: What is the max event rate before Central struggles?
2. **Policy Impact**: How much overhead does policy evaluation add?
3. **Scaling**: Does performance degrade linearly with event rate?
4. **Comparison**: How does this compare to process event handling?
5. **Database**: What is the alert storage rate (MB/sec)?

## Plotting Scripts

Three plotting scripts are provided to visualize test results:

### 1. plot-file-activity.py
Python script for comparing two metric files (without policy vs with policy).

Usage:
```bash
python3 plot-file-activity.py \
  results1/metrics_central_cpu.txt "Without Policy" \
  results2/metrics_central_cpu.txt "With Policy" \
  "Central CPU" "CPU Cores" \
  output.png
```

### 2. MakePlots-file-activity.sh
Generates all comparison plots for a single batch size.

Usage:
```bash
./MakePlots-file-activity.sh \
  <without_policy_dir> \
  <with_policy_dir> \
  <output_dir>
```

Generates 10 plots:
- CPU usage for central, central-db, sensor
- Memory usage for central, central-db, sensor
- Row counts for alerts, deployments tables
- Size in bytes for alerts, deployments tables

### 3. MakeAllPlots-file-activity.sh
Batch processes all test results.

Usage:
```bash
./MakeAllPlots-file-activity.sh [base_dir] [num_sensors] [run_time]
```

Default: `./MakeAllPlots-file-activity.sh perf 1 10m`

### 4. plot-batch-comparison.py
Advanced script showing how metrics scale with event rate.

Usage:
```bash
python3 plot-batch-comparison.py <base_dir> [output_dir]
```

Generates 6 scaling plots:
- Central CPU vs event rate
- Central memory vs event rate
- Central-DB CPU vs event rate
- Alert count vs event rate
- Alert size vs event rate
- Policy overhead percentage vs event rate

## Files Modified/Created

```
scale/dev/
├── perf-test-file-activity.sh          (new)
├── CheckDB-file-activity.sh            (new)
├── enable-file-activity-policy.sh      (new)
├── prometheus-query-file-activity.sh   (new)
├── run-file-activity-tests.sh          (new)
├── plot-file-activity.py               (new)
├── MakePlots-file-activity.sh          (new)
├── MakeAllPlots-file-activity.sh       (new)
├── plot-batch-comparison.py            (new)
├── FILE_ACTIVITY_TESTING.md            (new)
└── SUMMARY.md                          (this file)

scale/workloads/
├── file-activity-batch-10.yaml         (new)
├── file-activity-batch-50.yaml         (new)
├── file-activity-batch-100.yaml        (new)
├── file-activity-batch-250.yaml        (new)
└── file-activity-batch-500.yaml        (new)
```

## Typical Workflow

1. **Run tests** (automated):
   ```bash
   ./run-file-activity-tests.sh 1 10m
   ```

2. **Generate individual plots**:
   ```bash
   ./MakeAllPlots-file-activity.sh perf 1 10m
   ```

3. **Generate scaling comparison**:
   ```bash
   python3 plot-batch-comparison.py perf
   ```

4. **Review results**:
   ```bash
   ls -R perf/plots_*
   ls perf/comparison_plots/
   ```

---

**Ready to test!** Start with a single low-rate test to validate the setup.
