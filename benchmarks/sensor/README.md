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

## Correctness tests only (`go test`)

```bash
go test ./sensor/benchmark/... -count=1
```

Unit tests for scrape math, scorecard schema, scenario parsing, and PR comment compare — no Sensor process.

## Version bumps

- **v0:** change dev workload or phases → bump `metadata.version` in `scenario.yaml` and note in PR.
- **v1 (future):** changing `v1/workload.yaml` invalidates historical v1 comparisons → bump version or new scenario directory.

Design: [docs/superpowers/specs/2026-05-21-sensor-benchmark-design.md](../../docs/superpowers/specs/2026-05-21-sensor-benchmark-design.md)
