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
| `test-suite` | A Go package test process | Package-level interval, containing its Go test-case intervals. A cached result is a `skipped` marker, not a fabricated zero-duration span. |
| `post-test-stage` | A whole `PostClusterTest` or `FinalPost` run | Aggregate collection/processing time. Child collection commands are also emitted separately. |
| `post-test-collection` | Individual collection, artifact, or result-staging operations | Per-operation duration; includes command-level events from `RunWithBestEffortMixin` and named non-Groovy result/log collection calls. |

## Reading the data

- A span has matching `start` and `end` events with the same `span_id`; the end
  event carries `duration_ms` and `outcome`.
- `test-case` spans are nested in their enclosing task/lane spans. Go case spans
  may also be nested in a `test-suite` package span. These are inclusive intervals;
  overlapping or nested durations must not be added together.
- Infra-only mode emits explicit `skipped` markers for test execution and
  post-test collection. A skipped marker is not a zero-duration measurement.
- `post-test-stage` spans contain `post-test-collection` child spans. Do not add
  parent and child durations together.
- Different lanes run concurrently. Their wall-clock durations are not
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
