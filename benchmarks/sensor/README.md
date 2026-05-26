# Sensor benchmarks (`sensor-bench`)

## Scenario maturity

| Path | ID | Purpose |
|------|-----|---------|
| `scenarios/v0/steady-synthetic-dev/` | `steady-synthetic-dev-v0` | **Active for now.** Small workload, harness/CI development. Scorecards are **dev-only** — not comparable to v1. |
| `scenarios/v1/steady-synthetic/` | `steady-synthetic-v1` | **Planned** full benchmark (scale-like load). Not used until promoted after v0 calibration. |

Always check `metadata.name`, `metadata.version`, and `metadata.labels.maturity` in the scorecard / scenario YAML.

## Run locally (v0) — full benchmark

```bash
make sensor-bench
# or
go build -o bin/sensor-bench ./tools/sensor-bench
./bin/sensor-bench -scenario benchmarks/sensor/scenarios/v0/steady-synthetic-dev -out scorecard.json
```

This runs a real in-process Sensor. It is **not** invoked via `go test`.

**Timing (v0, version 1):** ~1 min sync + **180s** steady measurement + shutdown → about **4–5 min** wall time per run. The steady window is 180s (not 60s) so metrics average over several deployment/network/process cycles and back-to-back same-version runs typically agree within ~1%.

Do not compare scorecards with `scenario.version: "0"` (`measure_sec: 60`) to version `1`.

## Correctness tests only (`go test`)

```bash
go test ./sensor/benchmark/... -count=1
```

Unit tests for scrape math, scorecard schema, scenario parsing, and scorecard compare — no Sensor process.

## Compare two scorecards

Same `scenario.id`, `version`, and `maturity` required (e.g. two `steady-synthetic-dev-v0` **version 1** runs).

```bash
go build -o bin/sensor-bench ./tools/sensor-bench

# candidate (newer) vs baseline (reference)
./bin/sensor-bench \
  -compare-base scorecard-base.json \
  -compare-head scorecard-pr.json

# or write markdown to a file
./bin/sensor-bench \
  -compare-base scorecard-base.json \
  -compare-head scorecard-pr.json \
  -compare-out bench-compare.md
```

`-compare-head` is the run under test (PR head); `-compare-base` is the reference (merge-base or previous run).

## Version bumps

- **v0:** change dev workload or phases → bump `metadata.version` in `scenario.yaml` and note in PR.
- **v1 (future):** changing `v1/workload.yaml` invalidates historical v1 comparisons → bump version or new scenario directory.

Design: [docs/superpowers/specs/2026-05-21-sensor-benchmark-design.md](../../docs/superpowers/specs/2026-05-21-sensor-benchmark-design.md)
