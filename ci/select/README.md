# CI job selection

The selector enrolls four complete jobs in `unit-tests.yaml`: `go`,
`go-postgres`, `go-bench`, and `sensor-integration-tests`. It gates setup,
tests, and report publication at step level, keeping job names and matrices
present. Selecting `go` includes its integration and Operator tests.

Each job gates its workflow-level preamble and a suite-specific composite
action on the selection decision. The preamble stays at workflow level so
its post-job metrics hook can read its step outputs. It runs before suite
setup, including PostgreSQL startup.

JUnit generation and publication remain inside the suite with their status
conditions. Normal steps check the enclosing job as well as the composite's
implicit success state; this preserves failure and cancellation handling
before and during suite execution. Shell steps retain Ubuntu's default
`bash -e` pipeline behavior.

Jira reporting stays at job level so infrastructure failures are captured
using the completed suite's job status. Its gate also suppresses reports
when selection skips the suite.

On pull requests, each gate reads the `ci-dispatcher-enforce` label from
`GITHUB_EVENT_PATH`. Without it, selection only prints a preview and the job
retains its existing behavior. Removing the label restores that behavior,
including the UI-only outer filters and release-variant label. With
enforcement, the benchmark and Sensor outer filters admit the job so the
selector owns the decision. Pushes and tags retain their existing behavior.

Adding or removing `ci-dispatcher-enforce` restarts the Unit Tests workflow's
test work. Other label events skip change detection and its dependent jobs,
and use a separate concurrency group to protect active and pending runs.

The unconditional **CI selector tests** workflow validates configuration,
tests the selector, and prints a preview even when an enrolled job's legacy
outer filter prevents that job from starting. Job gates also write decisions
and reasons to logs and step summaries. Making selector validation a required
check requires a separate repository ruleset change.

## Policy

`ci/test-domains.yaml` lists the inventory and five `safe-paths` rules
followed by `remaining`. For each target, a nonempty diff is safe to skip
only when **every changed path** matches at least one rooted Go regular
expression in a rule explicitly listing that target in `skip`. Different
paths can be covered by different rules. Overlapping rules combine coverage,
independently of rule order. The explanation names all matching rules that
justify a skip. Only after collecting coverage does the final `remaining`
rule run every enrolled target still undecided.

| Changed paths | Jobs skipped |
| --- | --- |
| `ui/`, `docs/`, and root Markdown, in any combination | All four |
| Listed Central domains, optionally with root Markdown | Sensor integration |
| Listed Sensor domains only | PostgreSQL tests and benchmarks |
| `scanner/` and listed non-API Operator paths, alone or combined | PostgreSQL tests, benchmarks, Sensor integration |

Uncovered shared packages, generated files, protobufs, build scripts, and
workflow changes run all four, including when mixed with covered paths.
A Central change plus a listed Sensor change also runs all four because no
target has coverage for both. An empty diff runs everything through
`remaining`.

The schema accepts only `safe-paths` skips with explicit target names and
a final `when: remaining` rule with `run: ['*']`. Path-rule wildcard skips
are invalid: enrolling a target grants no existing skip permission. The
final wildcard deliberately covers undecided targets. Unknown fields,
invalid scalar types, nulls, duplicate names, unknown targets, invalid
regexes, and malformed catch-alls fail validation. There are no dependency
rules, label-based policy rules, suite filters, positive path-based run
rules, or run/skip conflict resolution.

The long-term goal is no unit-test execution for Markdown-only changes;
a lightweight planner may still run. This policy covers root Markdown and
the docs tree. Markdown inside a backend package is not a docs exemption.
Broader Markdown coverage needs a separate policy review of fixture inputs.
The workflow has no workflow-level path filter.

### Boundary evidence

The rules were checked against the test targets in the root `Makefile` on
2026-10-27, using `go list -deps -test -race` for Linux/amd64 with `test`
and `release,test`, adding `sql_integration` for PostgreSQL and benchmarks.
Discovery covered 285 PostgreSQL package directories and 64 benchmark
directories, with the Makefile's exclusions and nested module boundaries.
Sensor discovery covered `./sensor/tests/...`. All six graphs loaded without
package errors. The same checks on Darwin's default variant agreed.

The proposed `^operator/` boundary had a counterexample:
`pkg/k8sintrospect` imports `operator/api/v1alpha1`, making it a transitive
input to both PostgreSQL and Sensor tests. The Operator rule therefore lists
reviewed non-API roots and root files; API changes and new subdirectories
run all four. The benchmark graph
had no Operator imports. The other proposed skips had no matching package
inputs in the skipped jobs' graphs.

The frontend tree has no Go source files. The Go job's unit, integration,
and Operator test targets were inspected for inputs under `ui/`, `docs/`,
or root Markdown; no matching test or generation input was found.

Embedded inputs and cross-directory file references were also inspected.
Sensor tests use `tests/complianceoperator/crds`; other inspected references
use shared `pkg/` or `generated/` files. These paths run all four. No
additional counterexample was found. This evidence describes the current
dependency graph: review new imports and external file inputs when changing
these boundaries. There is no automated dependency audit; new imports or
external inputs can invalidate a boundary without these path tests detecting
it. Dependency discovery and periodic auditing are deferred.

Sensor integration uses fake Central/scan messages and the fixed test image
references in `sensor/tests/images-to-prefetch.txt`.

## Failure handling and trust

Git uses the event's base and head with
`diff --no-renames --name-only -z BASE...HEAD`. Both sides of renames
participate, and NUL separation preserves unusual filenames. PR checkouts
fetch full history; tests still use their existing checked-out revision.

Consumers run on unreadable diffs, missing refs, invalid rules, failed
builds, missing or crashing executables, and unknown targets. Error recovery
runs independently of the normal `remaining` policy and requires no readable
policy inventory. Configuration validation still fails on malformed input.

The composite action builds before dependency preparation, PostgreSQL, image
pulls, or Kind. A failed build cannot execute a stale binary. The shell gate
recovers execution failures with `action=run`. The import validator recognizes
`ci/select` as a build root because it checks every root-module Go package.
No import boundary is relaxed for another component.

Selection uses the ordinary `pull_request` event and its existing checkout
and secret handling. It does not use `pull_request_target`, API clients, PR
comments, Prow dispatch, or image/e2e selection. Image builds remain available
for consumers outside these four jobs, including later Prow `/test` requests.

## Local checks and rollout

From the repository root:

```sh
ci/select/build.sh
ci/select/bin/ci-select validate
printf '%s\n' ui/example.tsx README.md > /tmp/ci-selection-files
ci/select/bin/ci-select resolve --files-file /tmp/ci-selection-files
go test -race ./ci/select
go vet ./ci/select
.gotools/bin/golangci-lint run ./ci/select
.gotools/bin/actionlint .github/workflows/unit-tests.yaml .github/workflows/ci-select-tests.yaml
shellcheck ci/select/build.sh ci/select/gha-gate.sh
```

Building in each consumer still costs time and can require a toolchain
download. Prebuilt delivery is separate work. Skipping these jobs does not
guarantee a short whole-PR turnaround while other workflows run.

Before rolling out enforcement, inspect live Actions runs for a docs-only
skip, a partial selection, and an unmatched change. Toggle enforcement off
and check preview behavior, required check conclusions, downstream jobs,
and both Go matrix variants. Local tests and YAML validation cannot prove
GitHub's check aggregation behavior.
