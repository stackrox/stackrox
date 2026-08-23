# File Activity Performance Testing

This directory contains scripts for testing the performance and scale of fake file activity generation in StackRox.

## Overview

The file activity testing framework allows you to:
- Generate varying rates of file activity events (controlled via batch size)
- Compare performance with and without policy enforcement
- Monitor system resource usage (CPU, memory, database table sizes)
- Collect metrics and diagnostic bundles for analysis

## Key Differences from Process Baseline Testing

### Event Generation Patterns
- **File Activity**: Fixed global rate (one generator regardless of deployments)
  - Rate = batchSize / activityInterval
  - Example: batchSize=100, activityInterval=100ms → 1000 events/sec
  - Doubling deployments does NOT double event rate
  
- **Process Events**: Per-pod rate (scales linearly with deployments)
  - Each pod has its own generator
  - Doubling deployments DOES double event rate

### Database Tables Monitored

File activity tests monitor:
- `alerts` - File activity events may trigger policy violations creating alerts
- `deployments` - Container-level file activity is associated with deployments

Process baseline tests monitored:
- `process_indicators` - Individual process executions
- `process_baselines` - Baseline definitions per deployment
- `process_baseline_results` - Baseline lock status
- `alerts` - Process baseline violations

## Workload Configurations

Located in `scale/workloads/`, varying by batch size (all use 100ms interval):

| File | Batch Size | Events/Second | Description |
|------|------------|---------------|-------------|
| `file-activity-batch-10.yaml` | 10 | 100 | Low rate baseline |
| `file-activity-batch-50.yaml` | 50 | 500 | Medium-low rate |
| `file-activity-batch-100.yaml` | 100 | 1,000 | Medium rate (same as file-activity-scale.yaml) |
| `file-activity-batch-250.yaml` | 250 | 2,500 | High rate |
| `file-activity-batch-500.yaml` | 500 | 5,000 | Very high rate |

### Workload Parameters

Each workload includes:
- **Deployments**: 100 deployments with 3 pods each (2 containers per pod)
- **Nodes**: 10 fake nodes
- **Namespaces**: 3
- **File Activity**:
  - `activityInterval`: 100ms (fixed)
  - `batchSize`: varies (10, 50, 100, 250, 500)
  - `numPaths`: 500 unique file paths
  - `nodeEventPercent`: 50 (50% node-level, 50% container-level events)

## Scripts

### Main Test Script

**`perf-test-file-activity.sh`**
```bash
./perf-test-file-activity.sh <num_sensors> <run_time> <workload_name> <test_with_policy>
```

Example:
```bash
./perf-test-file-activity.sh 1 10m file-activity-batch-100 true
```

Parameters:
- `num_sensors`: Number of sensor instances (usually 1)
- `run_time`: How long to generate file activity events (e.g., 5m, 10m, 30m)
- `workload_name`: Name of workload config (without .yaml)
- `test_with_policy`: `true` or `false` - enable file activity policy

### Batch Test Runner

**`run-file-activity-tests.sh`**
```bash
./run-file-activity-tests.sh [num_sensors] [run_time]
```

Runs all batch size configurations (10, 50, 100, 250, 500) both with and without policy.

Example:
```bash
./run-file-activity-tests.sh 1 10m
```

### Supporting Scripts

- **`CheckDB-file-activity.sh`**: Queries database for file activity alert counts
- **`enable-file-activity-policy.sh`**: Creates a test policy that triggers on file activity
- **`prometheus-query-file-activity.sh`**: Collects Prometheus metrics for CPU, memory, and table sizes

## Test Workflow

1. **Deploy ACS** with selected workload configuration
2. **Wait for stabilization** (30 seconds)
3. **Generate events** for specified run_time
4. **Collect diagnostic bundle** #1
5. **Query database** for alert counts (if policy enabled)
6. **Wait 5 minutes** for processing
7. **Collect diagnostic bundle** #2
8. **Query database** again
9. **Run K6 load test** (optional performance testing)
10. **Wait 10 minutes** for final stabilization
11. **Collect Prometheus metrics** (CPU, memory, table sizes)

## Results

Results are stored in directories named:
```
perf/file_activity_results_<sensors>_<runtime>_<workload>_policy_<true|false>/
```

Each results directory contains:
- `diagnostic_bundle_1/` - Bundle after initial generation period
- `diagnostic_bundle_2/` - Bundle after 5 minute wait
- `file_activity_alerts_1.txt` - Alert counts (if policy enabled)
- `file_activity_alerts_2.txt` - Alert counts after wait (if policy enabled)
- `metrics/` - Prometheus time series data:
  - `*_central_cpu.txt` - Central CPU usage
  - `*_central_mem.txt` - Central memory usage
  - `*_central-db_cpu.txt` - Database CPU usage
  - `*_central-db_mem.txt` - Database memory usage
  - `*_sensor_cpu.txt` - Sensor CPU usage
  - `*_sensor_mem.txt` - Sensor memory usage
  - `*_alerts.txt` - Alerts table row count over time
  - `*_alerts_bytes.txt` - Alerts table size in bytes over time
  - `*_deployments.txt` - Deployments table row count
  - `*_deployments_bytes.txt` - Deployments table size

## Analysis Questions

### Performance Impact
1. How does file activity event rate affect Central CPU/memory usage?
2. What is the throughput limit before Central starts dropping events?
3. Does policy enforcement significantly impact performance?

### Database Growth
1. How fast does the alerts table grow with policy enabled?
2. Does alert generation rate correlate linearly with event rate?
3. What is the storage impact over time?

### Comparison with Process Events
1. Is file activity processing more or less resource-intensive than process events?
2. Can the system handle higher rates of file activity compared to process events?

## Policy Details

The test policy (`enable-file-activity-policy.sh`) creates a simple policy:
- **Name**: "File Activity Test Policy"
- **Event Source**: DEPLOYMENT_EVENT (runtime events)
- **Severity**: LOW_SEVERITY
- **Trigger**: Matches on process ancestors (broad matching for testing)

For production testing, you may want to create more specific policies that match:
- Specific file paths (e.g., `/etc/ssh`, `/etc/pam.d`)
- Specific file operations (create, delete, modify, permission change)
- Specific containers or deployments

## Tips

1. **Start with low batch sizes** (10-50) to establish baseline performance
2. **Monitor for dropped events** in sensor logs if you see gaps in metrics
3. **Compare with and without policy** to isolate policy enforcement overhead
4. **Use diagnostic bundles** to check for errors or warnings in component logs
5. **Run multiple iterations** at the same configuration to check for consistency

## Cleanup

After tests:
```bash
./TeardownTest.sh
kubectl delete ns stackrox stackrox1
```

## Future Enhancements

Potential improvements to the testing framework:
- [ ] Add plotting scripts to visualize results (similar to MakePlots.sh)
- [ ] Test with multiple policies active simultaneously
- [ ] Vary nodeEventPercent to test container vs node event handling
- [ ] Test with different numPaths values (path diversity impact)
- [ ] Add alerts table query to show alert growth rate over time
- [ ] Compare performance across different StackRox versions
