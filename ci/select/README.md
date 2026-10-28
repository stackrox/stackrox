# CI job selection

The selector enrolls four complete jobs in `unit-tests.yaml`: `go`,
`go-postgres`, `go-bench`, and `sensor-integration-tests`. It gates setup,
tests, and report publication at step level, keeping job names and matrices
present. Selecting `go` includes its integration and Operator tests.

Each job calls a suite-specific composite action behind one selection gate.
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

`ci/test-domains.yaml` lists the inventory and five file rules followed by
`remaining`. Each file rule requires a nonempty diff in which **every path**
matches at least one of that rule's rooted Go regular expressions. Matched
rules combine skips; `remaining` runs every target that has no skip.

| Changed paths | Jobs skipped |
| --- | --- |
| `ui/`, `docs/`, and root Markdown, in any combination | All four |
| Listed Central domains only | Sensor integration |
| Listed Sensor domains only | PostgreSQL tests and benchmarks |
| `scanner/` only | PostgreSQL tests, benchmarks, Sensor integration |
| Listed non-API Operator paths only | PostgreSQL tests, benchmarks, Sensor integration |

Mixed domains, shared packages, generated files, protobufs, build scripts,
and workflow changes run all four. A Central change plus `README.md` also
runs all four. Markdown inside a backend package is not a docs exemption.
An empty diff runs everything through `remaining`.

The schema accepts only file-rule skips and a final `remaining` rule with
`run: ['*']`. Unknown fields, invalid scalar types, nulls, duplicate names,
unknown targets, invalid regexes, and malformed catch-alls fail validation.
There are no dependency rules, label-based selection rules, suite filters,
run/skip conflict resolution, or separate run-all overrides.

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
these boundaries.

Sensor integration uses fake Central/scan messages and the fixed test image
references in `sensor/tests/images-to-prefetch.txt`.

## Failure handling and trust

Git uses the event's base and head with
`diff --no-renames --name-only -z BASE...HEAD`. Both sides of renames
participate, and NUL separation preserves unusual filenames. PR checkouts
fetch full history; tests still use their existing checked-out revision.

`ci/decision-defaults` independently declares an all-run fallback inventory.
Validation requires exactly the policy inventory and only `run` decisions.
Consumers run on unreadable diffs, missing refs, invalid rules or defaults,
failed builds, missing or crashing executables, and unknown targets. Recovery
also runs if the defaults themselves cannot be read. Logs record the cause;
configuration validation still fails on malformed input.

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
