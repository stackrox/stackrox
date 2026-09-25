# E2E timing events (MVP)

Timing events are one-line JSON records prefixed with `e2e_timing `. The initial
instrumentation is enabled by default only for the GHA GKE QA and non-Groovy
lanes. Set `E2E_TIMING_ENABLED` explicitly to override that default.

## Phases

| Phase | Scope | How to interpret it |
| --- | --- | --- |
| `test-lane-command` | The outer QA Part 1, QA Part 2, or non-Groovy script invocation | Inclusive lane-command time. It includes script setup and other work outside the narrower spans. |
| `test-build` | QA backend scanner-proto copy/generation and Gradle `assemble testClasses` prerequisites | Runner/build preparation, separate from the Gradle test-task span. |
| `test-execution` | QA BAT/full/sensor-bounce Gradle task groups; non-Groovy roxctl scripts, Bats, API, proxy, destructive, and external-backup suites | Suite-level duration, not per Groovy spec or Go test function. A suite invocation can include test-framework startup or compilation performed by that command. |
| `post-test-stage` | A whole `PostClusterTest` or `FinalPost` run | Aggregate collection/processing time. Child collection commands are also emitted separately. |
| `post-test-collection` | Individual collection, artifact, or result-staging operations | Per-operation duration; includes command-level events from `RunWithBestEffortMixin` and named non-Groovy result/log collection calls. |

## Reading the data

- A span has matching `start` and `end` events with the same `span_id`; the end
  event carries `duration_ms` and `outcome`.
- Infra-only mode emits explicit `skipped` markers for test execution and
  post-test collection. A skipped marker is not a zero-duration measurement.
- `post-test-stage` spans contain `post-test-collection` child spans. Do not add
  parent and child durations together.
- Different lanes run concurrently. Their wall-clock durations are not
  additive when estimating end-to-end PR turnaround.

## Current limits

- GHA provisioning before the outer command, and workflow steps outside the
  instrumented command/post-test helpers, are not covered by these spans.
- QA and Go suites are measured as task groups, not individual tests. Detailed
  per-test attribution requires consuming Gradle/JUnit and Go test reports.
- The rest of the GHA/Prow test matrix is not enabled by this MVP; missing
  events there mean “not instrumented,” not zero time.
