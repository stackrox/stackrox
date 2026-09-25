# GitHub Actions

## Optional store cache benchmark comparison

Add `ci-bench-tests` to a PR targeting master to run the **Store cache benchmark
comparison** workflow. It runs when that label is added and on subsequent pushes
or reopening while the label is present. Removing the label prevents future runs;
cancel an active run through Actions if needed. Unrelated label changes do not
restart or cancel the comparison. The normal one-iteration `go-bench` check is
unchanged.

The comparison builds the master parent of the tested PR merge and the merge
itself on one runner, then measures ten alternating rounds with the same Go
toolchain, PostgreSQL instance, and benchmark definitions. It covers cluster/namespace
authorization loading, deployment/pod reads, image export, and report queries.
Each process seeds fresh test data; generated IDs can differ. Stateful process
baseline evaluation and full report delivery are outside this comparison.

Results appear in the job summary and the `store-cache-benchmarks-<PR number>`
artifact, including raw observations, build logs, environment details, and
`benchstat` timing/allocation comparisons. Performance differences do not fail
the job; benchmark errors do. Failed runs also upload PostgreSQL server logs and
database/resource state to help diagnose timeouts.

For a local run against a dedicated PostgreSQL server, install the pinned
`benchstat` version from the workflow, then run:

```sh
scripts/ci/compare-store-cache-benchmarks.sh BASE_REF HEAD
```

`BENCHSTAT_BIN`, `BENCH_OUTPUT_DIR`, `BENCH_ROUNDS` (default `10`), and `BENCHTIME`
(default `1s`) override the tool, output directory, sample count, and measurement
duration. A smoke run can use `BENCH_ROUNDS=1 BENCHTIME=1x`. Benchmark sources are
taken from the candidate commit, so commit benchmark changes before comparing.

## Upstream Release Automation

### General Rules

* **Ensure reentrancy**: expect a workflow to be re-run from failed state;
* **Suggest the next step** if it is not automated: print `:arrow_right:` emoji
  with the suggested action to `$GITHUB_STEP_SUMMARY` or post a message to Slack;
* **Highlight major things**: print to stdout with `::error::`, `::warning::` or
  `::notice::` prefix to higlight important status. Markdown is not supported;
* **Log minor things**: print to `$GITHUB_STEP_SUMMARY` with markdown to describe
  the executed actions and suggest the next step;
* **Support dry-run**: use the `DRY_RUN` environment variable to check for dry-run,
  it holds `true` or `false` values;
* **Extract large scripts**: look in the `scripts` folder for examples;
