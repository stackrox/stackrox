# steady-synthetic-v1 (planned — not active yet)

This directory holds the **target v1** scale profile (derived from `scale/workloads/default.yaml`).

It is **not** used for initial harness development or CI. Use **`v0/steady-synthetic-dev`** until v1 is calibrated and explicitly promoted.

When activating v1:

1. Tune `workload.yaml` and phase timings from v0 CI experience.
2. Bump `scenario.yaml` `metadata.version`.
3. Document the cutover in `benchmarks/sensor/README.md`.
