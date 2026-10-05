# E2E timing events (MVP)

Timing events are one-line JSON records prefixed with `e2e_timing `. The initial
instrumentation is enabled by default only for the GHA GKE QA and non-Groovy
lanes. Set `E2E_TIMING_ENABLED` explicitly to override that default.

## Phases

| Phase | Scope | How to interpret it |
| --- | --- | --- |
| `test-lane-command` | The outer QA Part 1, QA Part 2, or non-Groovy script invocation | Inclusive lane-command time. It includes script setup and other work outside the narrower spans. |
| `test-build` | QA backend scanner-proto copy/generation and Gradle `assemble testClasses` prerequisites | Runner/build preparation, separate from the Gradle test-task span. |
| `test-execution` | One event per QA Gradle test task (`testParallel`, `testRest`, `testParallelBAT`, `testBAT`, `testSensorBounce`, `testSensorBounceNext`); non-Groovy roxctl scripts, Bats, API, proxy, destructive, and external-backup suites | Inclusive command/task timing, including its runner overhead. It is a parent/context span, not a per-test measurement. |
| `test-case` | Individual Groovy/Spock cases and Go test functions/subtests | Case-level execution intervals. Groovy names include the Gradle task and spec class; Go names include the package and hierarchical test name. Concurrent cases may overlap. |
| `test-suite` | A Groovy/Spock test class or Go package test process | Suite-level interval, containing its test-case intervals. Groovy class spans include class fixtures such as `setupSpec`/`cleanupSpec`; Go package spans include package setup. A cached Go result is a `skipped` marker, not an executed test interval. |
| `test-activity` | Selected shared test helpers in Groovy and Go | Short spans for K8s fixture lifecycle/readiness and StackRox deployment/violation visibility. They overlap cases; do not add them to lane totals. |
| `post-test-stage` | A whole `PostClusterTest` or `FinalPost` run | Aggregate collection/processing time. Child collection commands are also emitted separately. |
| `post-test-collection` | Individual collection, artifact, or result-staging operations | Per-operation duration; includes command-level events from `RunWithBestEffortMixin` and named non-Groovy result/log collection calls. |

## Reading the data

- A span has matching `start` and `end` events with the same `span_id`. Both
  carry UTC timestamps; consumers derive elapsed time from `end - start`. The
  end event carries the outcome. Keeping timestamps as the sole time source
  avoids storing a second, potentially inconsistent duration value.
- `test-case` spans are nested in their enclosing task/lane spans. Groovy cases
  and Go cases may also be nested in `test-suite` spans. These are inclusive
  intervals; overlapping or nested durations must not be added together.
- `test-activity` spans annotate selected shared helpers inside individual
  cases. Go helper markers are normalized by `go_test_timing.py` using the
  associated `go test -json` timestamps; the raw marker is hidden from normal
  test output. Activity time overlaps its test case and is for attribution,
  not another duration to add to the lane total.
- Infra-only mode emits explicit `skipped` markers for test execution and
  post-test collection. A skipped marker is not a measured interval.
- `post-test-stage` spans contain `post-test-collection` child spans. Do not add
  parent and child elapsed times together.
- Different lanes run concurrently. Their elapsed times are not
  additive when estimating end-to-end PR turnaround.

## Current limits

- GHA provisioning before the outer command, and workflow steps outside the
  instrumented command/post-test helpers, are not covered by these spans.
- Go and Groovy execution spans are emitted only when timing is enabled; normal
  test output is retained. Build/setup work inside a test command remains part
  of its enclosing command span rather than being assigned to an individual case.
- Go case-level events are collected for test commands routed through
  `scripts/go-test.sh`; direct `go test` invocations are not intercepted. The
  raw `test.log` keeps these records; JUnit conversion filters them out.
- The rest of the GHA/Prow test matrix is not enabled by this MVP; missing
  events there mean “not instrumented,” not zero time.

## Groovy parallelization audit

The GHA GKE QA lane also enables an observer-only Groovy parallelization audit.
It records spec start/end markers, unique Kubernetes HTTP method/path pairs,
Central gRPC service/method names, and changes to process-global Central auth
configuration. It never records request bodies, headers, query strings, tokens,
or credentials. Repeated identical operations within a spec/feature are
deduplicated to keep log volume down.

When the audit is enabled, `run_timed.py` mirrors only `e2e_timing` and
`e2e_parallel_audit` records to `ARTIFACT_DIR/e2e-events.jsonl`, while retaining
the normal console output. The existing post-test artifact upload stores this
file in GCS alongside the JUnit reports. Prefer this raw JSONL file for analysis;
the console log remains a fallback for older runs and may not preserve valid
JSON delimiters.

To summarize a completed GHA run, stream the GCS event file through the analyzer:

```sh
gcloud storage cat gs://stackrox-ci-artifacts/stackrox/stackrox/RUN_ID-ATTEMPT/gke-qa-e2e-tests/part-1/junit-reports/e2e-events.jsonl \
  | python3 qa-tests-backend/scripts/e2e_parallelization_audit.py
```

Use `--format json` for structured output. The report compares spec footprints
within a run/lane. Each shared-resource finding says whether the spec lifetimes
overlapped, were sequential, or could not be classified; sequential findings
are candidates to investigate before parallelizing that pair. It is not a
parallel-safety certification: direct REST clients, external services,
controller side effects, and async operations that lose the test thread's MDC
context are not fully attributed. Central API conflicts are service-level
because request payloads are intentionally not observed. No reported conflict
means only that this run did not observe one in the instrumented paths.
