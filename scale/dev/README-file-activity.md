# File Activity Performance Testing - Quick Reference

Quick commands for running file activity performance tests. The harness is
config-driven: one JSON run config describes the load type, the rates and the
policies, and the same scripts drive both the **fake** (sensor built-in generator)
and **berserker** (DaemonSet) workloads.

## Quick Start

```bash
# Full matrix (every rate x every policy) for a run config
./run-perf-tests.sh configs/berserker-file-activity.json
./run-perf-tests.sh configs/fake-file-activity.json

# A single cell: <config> <rate> <policy-nickname>
./perf-test-cell.sh configs/berserker-file-activity.json 1000 alerting

# Plots (point at the run's results dir, which holds run-config.json)
./make-all-plots.sh perf/berserker-file-activity              # per-rate, one line per policy
python3 plot-batch-comparison.py perf/berserker-file-activity # scaling across rates
```

## Run config (`configs/*.json`)

```json
{
  "name": "berserker-file-activity",
  "loadType": "berserker",                 // berserker | fake
  "loadTemplate": "load-templates/berserker-file-activity.ber.tmpl",
  "rateField": "__RATE__",                 // placeholder substituted in the template
  "numSensors": 1,
  "runTime": "10m",
  "numDeployments": 1,                     // berserker: concurrent deployments (sharding)
  "resultsBaseDir": "perf",
  "rates": [
    { "value": 16.67, "rate": 100 },       // value -> __RATE__; rate = actual events/sec
    { "value": 166.67, "rate": 1000 }
  ],
  "policies": [
    { "nickname": "baseline",  "file": null },                      // no policy
    { "nickname": "alerting",  "file": "policies/berserker-data.json" }
  ]
}
```

- Each `rates` entry pairs the template substitution `value` with the actual total
  `rate` (events/sec). The `rate` identifies the cell and is the plot x-axis; the
  `value` is what gets `sed`-substituted for `rateField` in the load template.
- Each `policies` entry maps a `nickname` to a policy body JSON (`policies/*.json`),
  or `null` for the baseline/no-policy cell. The policy's File Path drives what the
  `fact` agent monitors.
- Add a rate or a policy by editing JSON — no shell changes.

## Templates and policies

- `load-templates/*.tmpl` — the load definition with a `__RATE__` placeholder
  (`.ber` for berserker, `.yaml` for fake). Rendered per rate into a generated
  workload file that the existing deploy scripts consume.
- `policies/*.json` — policy bodies POSTed to Central by `enable-policy.sh`.

## Scripts

| Script | Role |
|--------|------|
| `run-perf-tests.sh <config>` | Orchestrator: iterate policies x rates, resume-skip existing cells |
| `perf-test-cell.sh <config> <rate> <policy>` | Single cell: render load, deploy, measure, collect |
| `enable-policy.sh <policy.json>` | POST a policy body to Central |
| `lib/common.sh` | Shared helpers (`pt::*`) |
| `lib/load-{berserker,fake}.sh` | Load drivers (`load::*`) |
| `make-all-plots.sh <run_dir>` | Per-rate comparison plots (one line per policy) |
| `plot-batch-comparison.py <run_dir>` | Scaling comparison across rates |

## Results layout

```
perf/<name>/
├── run-config.json                  # copy of the config (plotters read this)
├── rate-100__baseline/
├── rate-100__alerting/
├── rate-1000__baseline/
├── ...
├── plots_rate-100/                  # make-all-plots.sh output
└── comparison_plots/                # plot-batch-comparison.py output
```

Each cell dir contains `diagnostic_bundle_{1,2}/`, `metrics/`, `k6_load_test.txt`,
`file_activity_alerts_{1,2}.txt` (when a policy is active) and, for berserker,
`berserker_ready_timestamp.txt`.

## Resume

`run-perf-tests.sh` skips any cell whose results dir already exists, so an
interrupted run restarts by re-invoking with the same config. Delete a cell dir to
force it to re-run.

## Full Documentation

See `FILE_ACTIVITY_TESTING.md` for comprehensive documentation.
