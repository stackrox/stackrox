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

## Config-Driven Harness

The harness is driven by a JSON run config (`configs/*.json`) that declares the load
type, the rates, and the policies. The same scripts run both the **fake** and
**berserker** workloads; see `README-file-activity.md` for the config schema.

## Scripts

### Orchestrator

**`run-perf-tests.sh`** — runs the full matrix (every policy x every rate) for a config:
```bash
./run-perf-tests.sh configs/fake-file-activity.json
./run-perf-tests.sh configs/berserker-file-activity.json
```
It copies the config to `<resultsBaseDir>/<name>/run-config.json`, and **skips any
cell whose results dir already exists** so an interrupted run resumes on re-invoke.

### Single Cell

**`perf-test-cell.sh`** — runs one (rate, policy) cell:
```bash
./perf-test-cell.sh <config.json> <rate> <policy_nickname>
# e.g.
./perf-test-cell.sh configs/fake-file-activity.json 1000 policy
```
- `rate`: the actual event rate of one of the config's `rates` entries (e.g. `1000`);
  the paired `value` is substituted into the load template's `__RATE__` field.
- `policy_nickname`: one of the config's `policies[].nickname` entries (`baseline`/
  `nopolicy` run with no policy).

### Supporting Scripts

- **`enable-policy.sh <policy.json>`**: POSTs a policy body (`policies/*.json`) to Central
- **`CheckDB-file-activity.sh`**: Queries database for file activity alert counts
- **`prometheus-query-file-activity.sh`**: Collects Prometheus metrics for CPU, memory, and table sizes
- **`lib/common.sh`**, **`lib/load-{berserker,fake}.sh`**: shared helpers and load drivers

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

Results are stored under a per-run directory, one cell dir per (rate, policy):
```
perf/<name>/
├── run-config.json                      # copy of the config (plotters read this)
├── rate-<rate>__<policy_nickname>/      # one cell, e.g. rate-1000__alerting
└── ...
```

Each cell directory contains:
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

Policies are standalone JSON bodies under `policies/`, referenced by nickname from a
run config and POSTed to Central by `enable-policy.sh`:
- `policies/fake-dirs.json` — "File Activity Test Policy", watches the fake-workload
  directories (`/etc/ssh/**`, `/etc/pam.d/**`, ...).
- `policies/berserker-data.json` — watches `/tmp/data/**` (where berserker writes).
- `policies/berserker-idle.json` — watches `/tmp/quiet/**` (no berserker activity).

All are RUNTIME / DEPLOYMENT_EVENT / LOW_SEVERITY. The policy's **File Path** field is
what drives file-activity monitoring: Sensor extracts it from enabled runtime
file-access policies and pushes it to the `fact` agent. Add a policy by dropping a new
JSON body in `policies/` and referencing it from a config. NOTE: File Path matches by
doublestar GLOB, not regex — use `/etc/ssh/**`, not `/etc/ssh/.*`.

## Visualizing Results

The plotters are config-driven: point them at a run's results dir
(`<resultsBaseDir>/<name>`), which holds `run-config.json`.

### Per-Rate Comparison Plots

One plot set per rate, overlaying one line per policy:

```bash
./make-all-plots.sh perf/berserker-file-activity
```

This generates comparison plots for CPU, memory, table sizes, event drops, etc.

### Scaling Comparison Plots

See how metrics scale with the event rate (one line per policy):

```bash
python3 plot-batch-comparison.py perf/berserker-file-activity
```

This creates plots showing:
- CPU/memory usage vs event rate
- Alert count and size vs event rate
- Policy enforcement overhead percentage
- Scaling trends across all rates

All plots are saved as PNG files with 150 DPI resolution.

### Summary Tables

Config-driven tables (one column group per policy, read from `run-config.json`):

```bash
python3 memory-table.py      perf/berserker-file-activity   # avg memory per component
python3 events-rate-table.py perf/berserker-file-activity   # observed vs configured event rate
python3 restart-oom-table.py perf/berserker-file-activity   # restarts and OOM-kills
python3 drops-table.py       perf/berserker-file-activity   # event drops per pipeline stage
```

Each takes `<run_dir> [rate1 rate2 ...]` and defaults to all rates in the config.

## Tips

1. **Start with low batch sizes** (10-50) to establish baseline performance
2. **Monitor for dropped events** in sensor logs if you see gaps in metrics
3. **Compare with and without policy** to isolate policy enforcement overhead
4. **Use diagnostic bundles** to check for errors or warnings in component logs
5. **Run multiple iterations** at the same configuration to check for consistency
6. **Generate plots after each test run** to visualize trends immediately

## Cleanup

After tests:
```bash
./TeardownTest.sh
kubectl delete ns stackrox stackrox1
```

## Future Enhancements

Potential improvements to the testing framework:
- [ ] Test with multiple policies active simultaneously
- [ ] Vary nodeEventPercent to test container vs node event handling
- [ ] Test with different numPaths values (path diversity impact)
- [ ] Add real-time alert generation rate monitoring
- [ ] Compare performance across different StackRox versions
- [ ] Add automated regression detection in plots
