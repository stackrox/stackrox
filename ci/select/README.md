# CI job selection

CI job selection decides which test suites a pull request can safely skip
based on the files it changes. For example, a change to `README.md` does not
need the backend test suites. A change to a shared Go package does.

The policy lives in [`ci/test-selection-rules.yaml`](../test-selection-rules.yaml). It lists
the test jobs covered by selection and the paths that are safe to change
without running each job. A **target** is one of these job names:

| Target | Tests controlled by this decision |
| --- | --- |
| `go` | Go unit tests, integration unit tests, and Operator integration and Helm tests |
| `go-postgres` | Go tests that require PostgreSQL |
| `go-bench` | PostgreSQL benchmarks, run once each to check that they compile and execute |
| `sensor-integration-tests` | Sensor integration tests |
| `e2e-byodb-tests` | Bring-your-own-database e2e tests |
| `e2e-db-backup-restore-tests-gke`, `e2e-db-backup-restore-tests-gke-konflux` | Database backup and restore e2e tests |
| `e2e-gke-upgrade-tests` | Central, PostgreSQL, and Sensor upgrade tests |
| `e2e-nongroovy-tests-gke`, `e2e-nongroovy-tests-gke-konflux` | Non-Groovy e2e tests |
| `e2e-qa-tests-gke`, `e2e-qa-tests-gke-konflux`, `e2e-qa-tests-ocp-fips-konflux` | Backend QA e2e tests |
| `e2e-ui-tests-gke`, `e2e-ui-tests-gke-konflux` | UI e2e tests |

Each target gets a `run` or `skip` decision for its whole suite. A `skip`
suppresses setup, tests, and reports while keeping the job check visible.
A `run` leaves the suite enabled, subject to its usual workflow conditions.
Decisions apply to every matrix variant of the target, including release
builds when those are enabled.

On GitHub pull requests, add the `ci-dispatcher-enforce` label to apply the
skip decisions. Without the label, selection prints a preview and the
workflow keeps its existing behavior. Errors in selection keep tests
enabled. The selector does not infer dependencies from the changed code;
the path rules record boundaries that have been reviewed.

## Read a policy file

This reduced example uses two targets and three rules:

```yaml
version: 1
jobs: [go, sensor-integration-tests]
rules:
  # Documentation outside test fixtures is safe without either test suite.
  - name: documentation
    when: safe-paths
    paths: ['^docs/', '^[^/]+\.md$', '!^docs/testdata/']
    skip: [go, sensor-integration-tests]

  # Central access-control changes are safe to change without Sensor tests.
  - name: central-access-control
    when: safe-paths
    paths: ['^central/sac/']
    skip: [sensor-integration-tests]

  # Run any job whose safe-path rules did not cover the whole diff.
  - name: remaining
    when: remaining
    run: ['*']
```

The fields have these meanings:

| Field | Meaning |
| --- | --- |
| `version` | The policy format version. Currently only `1` is supported. |
| `jobs` | The inventory of target names that this policy can decide for. Listing a name does not create a workflow job. |
| `name` | A unique, nonempty rule name used in explanations. The name does not select the rule's behavior. |
| `when` | The rule's behavior. The only supported values are `safe-paths` and `remaining`, explained next. |
| `paths` | Rooted Go regular expressions matching repository-relative paths. Prefix an entry with `!` to exclude matches from this rule. Required for `safe-paths`; omitted for `remaining`. |
| `skip` | Target names safe to skip for the listed paths. Required for `safe-paths`; omitted for `remaining`. |
| `run` | The final rule's action. Only `run: ['*']` on `remaining` is supported. |

### `when: safe-paths`

A `safe-paths` rule says: **these paths do not require the jobs named in
`skip`**. It contributes coverage for those jobs. It does not immediately
skip them just because one file matched.

For each job, the selector collects the paths covered by all `safe-paths`
rules explicitly naming that job in `skip`. It skips the job only if the
diff is nonempty and every changed path is covered. Different files may
be covered by different rules. Rules may overlap, and their order does not
affect the result.

In the example, `docs/guide.md` is safe for both jobs, but
`central/sac/example.go` is safe only for Sensor integration tests. Changing
both files skips Sensor integration tests and leaves Go tests to the final
rule. Adding `pkg/shared.go` leaves both jobs to the final rule because no
rule covers that file.

Patterns are regular expressions, not shell globs. After an optional `!`
prefix, each pattern must start with `^`, which anchors it to the repository
root. A path must match at least one inclusion and none of the exclusions:

- `'^docs/'` matches files anywhere under `docs/`.
- `'^[^/]+\.md$'` matches Markdown files in the repository root, such as
  `README.md`. It does not match `pkg/testdata/fixture.md`.
- `'^central/sac/'` matches files under that specific Central package.
- `'!^docs/testdata/'` removes files under `docs/testdata/` from this rule's
  coverage, even though `'^docs/'` also matches them.

Inclusion and exclusion order does not matter. Each rule must contain at
least one inclusion. Exclusions affect only their own rule: another rule
can still cover the path for the targets named in its own `skip` list.
They do not force a global `run` decision. Quote `!` entries so YAML treats
them as strings.

The YAML examples use single quotes so `\.` reaches the regular expression
engine as a literal dot. Use `$` when a pattern must also match the end of
the path.

Every `skip` entry must explicitly name a target in `jobs`. Wildcard skips
are rejected. If you add `new-check` to `jobs` without adding it to any
rule's `skip` list, existing rules grant it no skip permission.

### `when: remaining`

The `remaining` rule says: **run every enrolled job that safe-path coverage
did not already decide to skip**. It must appear exactly once, as the last
rule, and must have `run: ['*']` with no `paths` or `skip` fields. Here `*`
means all still-undecided targets in `jobs`.

With the example policy:

- Changing only `docs/guide.md` skips both jobs; none remain to run.
- Changing only `central/sac/example.go` skips Sensor integration tests;
  `remaining` runs Go tests.
- Changing `pkg/shared.go` runs both jobs through `remaining`.
- An empty diff also runs both jobs. An empty set of changed files never
  qualifies for a safe-path skip.

`safe-paths` and `remaining` are the only `when` values supported today.
Additional `when` conditions are planned for future versions.

Path rules cannot select `run`, and the final rule cannot select `skip`.
Conditions based on labels, imports, or dependencies are not part of this
policy format. The GitHub enforcement label controls whether to apply a
computed decision; it is separate from the rule language.

`remaining` defines normal policy behavior. Error recovery is separate:
invalid configuration, an unavailable planner, and requests for unknown
targets run tests even if the policy cannot be read.

## Combine rules with set operations

You can read the policy as sets of jobs safe to skip. Each rule covers its
included paths **minus** (`−`) its excluded paths. For each changed
path, take the **union** (`∪`) of the `skip` lists of all rules covering
that path. Then take the **intersection** (`∩`) of those sets across all
changed paths. A job survives that intersection only if every changed
path permits skipping it.

Looking at the four backend targets, suppose a PR changes
`central/sac/example.go` and `README.md`:

```text
Safe to skip for central/sac/example.go:
  A = {sensor-integration-tests}

Safe to skip for README.md:
  B = {go, go-postgres, go-bench, sensor-integration-tests}

Safe to skip for the complete diff:
  A ∩ B = {sensor-integration-tests}

Jobs left for remaining to run:
  jobs − (A ∩ B) = {go, go-postgres, go-bench}
```

The two files match different rules, but both allow Sensor integration
tests to be skipped. If you also change `pkg/shared.go`, its set of safe
skips is empty (`∅`), so `A ∩ B ∩ ∅ = ∅`: all four backend jobs run.
The e2e targets also run for either diff because it includes a Go file.

If two rules match the same file, their skip sets are combined with union
before that intersection. For example, skip sets `{go-postgres}` and
`{go-bench}` for one path give `{go-postgres, go-bench}` for that path.
Neither rule takes priority over the other.

For a nonempty diff `D`, the calculation is:

```text
covered(rule) = included paths − excluded paths
safe(p) = union of skip lists from rules whose covered paths contain p
skipped = intersection of safe(p) for every p in D
remaining = jobs − skipped
```

For an empty diff, `skipped` is explicitly empty and `remaining` runs all
jobs.

## Understand the current rules

The full policy has six `safe-paths` rules and one final `remaining` rule.
The table shows representative diffs and their results:

| Changed paths | Jobs skipped |
| --- | --- |
| UI files, optionally with eligible Markdown/text documentation | The four backend targets |
| Only `.md` and `.txt` files, outside the documented test/build exclusions | All fifteen targets |
| Non-Markdown/non-TXT files under `docs/` | None |
| Listed Central domains, optionally with root Markdown | Sensor integration |
| Listed Sensor domains only | PostgreSQL tests and benchmarks |
| `scanner/` and listed non-API Operator paths, alone or combined | PostgreSQL tests, benchmarks, Sensor integration |
| Listed Central paths plus listed Sensor paths | None |
| Any uncovered shared code path, including when mixed with covered paths | None |
| No changed files | None |

See each rule's comment and patterns in
[`ci/test-selection-rules.yaml`](../test-selection-rules.yaml) for its exact scope. Names
such as `scanner-only` identify rules in the logs; they do not prevent
coverage from combining with other rules.

Shared Go packages, generated code, protobufs, build scripts, and workflow
changes have no skip coverage and run every target. Operator API types
are shared inputs, so changes to them also run every target. New Operator
subdirectories need review before they join the rule's explicit path list.

The `documentation-only` rule matches `.md` and `.txt` extensions in any
directory, ignoring case, except for test/build inputs listed in its
`paths` exclusions. For example, `README.md` plus `sensor/notes.TXT` skips
all fifteen targets: the four backend targets and eleven e2e targets.
A lightweight planner and the independent selector checks still run.
The workflow has no workflow-level path filter.

The three `images-to-prefetch.txt` lists control images used by Sensor and
e2e tests. Other exclusions cover `testdata/` and `fixtures/` directories,
roxctl Bats expected output, QA test-image build contexts, generator
boilerplates, and Python requirements files. For example,
`roxctl/netpol/connectivity/diff/testdata/acs-security-demos-new-version/diff_output.md`
is expected test output, rather than documentation. These inputs do not
receive coverage from the documentation rule. Other reviewed domain rules
can still apply: changing `sensor/tests/images-to-prefetch.txt` keeps Go
and Sensor integration tests enabled while the Sensor rule permits skipping
PostgreSQL tests and benchmarks.

Add `ui/src/App.tsx` or `pkg/example.go` to a documentation diff, and every
e2e target runs under its existing eligibility rules. The UI rule still
allows the four backend targets to skip UI changes. An empty diff never
grants a skip. New test/build inputs with documentation extensions must
be reviewed and added to the exclusions when necessary.

Validation rejects unknown fields or targets, invalid scalar types, nulls,
duplicate names, invalid regular expressions, and malformed final rules.

## How GitHub Actions uses the result

Unit Tests and e2e Dispatch each call `ci-selection.yaml` once, outside their
test matrices. This reusable `workflow_call` workflow checks out the PR
history, builds the
selector through the shared `ci-select` action, and returns a compact JSON
`decisions` map through workflow outputs. Each enrolled unit-test job reads
that map without rebuilding or resolving again. Independent workflow runs compute
their own decisions.

Suite steps live directly in each job. The workflow-level preamble runs
before suite setup, including PostgreSQL startup, so its post-job metrics
hook can read its step outputs. Normal steps retain the implicit `success()`
condition. JUnit generation and publication retain `always()`; benchmark
diagnostics require failure without cancellation; Jira reporting requires
`!cancelled()` and observes the final job status. Every setup, test, and
reporting step also checks selection. Shell steps retain Ubuntu's explicit
`bash -e` pipeline behavior.

On pull requests, the planner reads the `ci-dispatcher-enforce` label from
`GITHUB_EVENT_PATH`. Without it, the output map applies `run` while logs
and summaries preview the policy decisions. Removing the label restores
existing behavior, including the UI-only outer filters and release-variant
label. With enforcement, the benchmark and Sensor outer filters admit the
job so selection owns the decision. Pushes and tags retain their existing
behavior.

Adding or removing `ci-dispatcher-enforce` reevaluates Unit Tests and e2e
Dispatch. Other label events skip planning, change detection, and test jobs.
Unit Tests uses a separate concurrency group for those ignored events to
protect active and pending runs.

### E2e jobs and required checks

The e2e dispatcher validates the applied map and forwards one `ci-action`
(`run` or `skip`) through each variant's existing reusable workflows. Each
suite job starts with a small Bash decision step. A selected skip disables
checkout, setup, provisioning, tests, cleanup, and reporting while the job
finishes successfully. Existing check names and workflow nesting stay the
same. The PostgreSQL upgrade job also starts for a selected skip, even
though its version-check step did not run.

For an enforced docs-only diff, the dispatcher admits all eleven variants,
including optional and Konflux variants without their usual labels, so
their checks can finish successfully. It bypasses unneeded image-wait
dependencies and gates image-wait steps when every consumer of that image
family skips. It does not suppress image builds.

When a target runs, its existing draft, UI-only, optional-label, and Konflux
eligibility rules still apply. Missing or malformed maps and planner
failure restore those existing rules; they cannot suppress an eligible
suite. Fork restrictions remain in force. Pushes, tags, and manual e2e
workflow dispatches retain their existing test behavior. Setup/test failures
and cancellation retain each step's original status conditions, combined
with the selection gate.

The unconditional **CI selector tests** workflow validates configuration,
tests the selector, and prints a preview even when an enrolled job's legacy
outer filter prevents that job from starting. The planner writes decisions
and coverage reasons to logs and step summaries; consumers log the applied
action. Making selector validation a required check needs a separate
repository ruleset change.

## Dependency boundaries

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
participate, and NUL separation preserves unusual filenames. The planner
fetches full history; test jobs keep their existing checkout revision and
need no extra history for selection.

Consumers run on unreadable diffs, missing refs, invalid rules, failed
builds, missing or crashing executables, and unknown targets. Error recovery
runs independently of the normal `remaining` policy and requires no readable
policy inventory. Configuration validation still fails on malformed input.

The planner builds before dependency preparation, PostgreSQL, image pulls,
or Kind. A failed build cannot execute a stale binary. The shell gate
recovers planner execution failures with `decisions={}`. Consumers parse
that output with `jq` in Unit Tests and the existing Python dispatcher in
e2e Dispatch, accepting only one JSON object with `run`/`skip`
values. Only an explicit `skip` for the consumer from a successful planner
suppresses work. Missing or malformed maps and planner failure run tests.
Explicit job status conditions bypass GitHub's default dependency-success
check while the event guards still reject unrelated label events and
cancellation. Legacy benchmark/Sensor UI-only filters apply when planning
succeeds without enforcement; planner failure admits these tests as well.
See GitHub's [status check functions](https://docs.github.com/en/actions/reference/workflows-and-actions/expressions#status-check-functions)
and [reusable workflow outputs](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#using-outputs-from-a-reusable-workflow).

The import validator recognizes `ci/select` as a build root because it
checks every root-module Go package. No import boundary is relaxed for
another component.

Selection uses the ordinary `pull_request` event and its existing checkout
and secret handling. It does not use `pull_request_target`, PR comments, or
Prow dispatch. The e2e dispatcher retains its existing read of PR labels.
Image builds remain available for other consumers, including later Prow
`/test` requests. E2e workflows outside the dispatcher's eleven targets,
such as Scanner functional tests, are not enrolled.

## Local checks and rollout

The tests use Go, Git, Bash, `jq`, and Python 3. The Python cases execute
the dispatcher's existing script with a local `gh` stub and make no API calls.

From the repository root:

```sh
ci/select/build.sh
ci/select/bin/ci-select validate
printf '%s\n' ui/example.tsx README.md > /tmp/ci-selection-files
ci/select/bin/ci-select resolve --files-file /tmp/ci-selection-files
go test -count=1 -race ./ci/select
go vet ./ci/select
.gotools/bin/golangci-lint run --enable=unused ./ci/select
.gotools/bin/actionlint .github/workflows/unit-tests.yaml .github/workflows/ci-select-tests.yaml .github/workflows/ci-selection.yaml
.gotools/bin/actionlint -config-file .github/actionlint.yaml .github/workflows/e2e-*.yaml
shellcheck --norc -P SCRIPTDIR -x ci/select/build.sh ci/select/gha-gate.sh ci/select/gha-consume.sh
```

Go tests cover policy decisions, Git diff handling, and shell recovery.
Workflow contract tests check required commands, decision wiring, ordering,
and selection/status guards without fixing the entire workflow layout.

Planning once per caller still requires a build and may download a Go
toolchain. There is no cross-run artifact or decision cache. Skipping these
jobs does not guarantee a short whole-PR turnaround while other workflows
run.

### Validation limits and possible follow-up

The Go workflow tests inspect YAML and execute the e2e dispatcher's Python
and the Bash decision steps with local fixtures. They do not execute
GitHub's `if` expressions or reproduce its scheduler. These tests verify
missing/malformed decisions and planner failure recovery, but cannot prove
that GitHub starts consumers after a failed or skipped `needs` dependency.

Before rolling out enforcement, inspect live Actions runs for these cases:

- A docs-only skip, a partial selection, and an uncovered change, including
  both Go matrix variants and successful checks for all eleven e2e variants.
  Confirm that docs-only e2e checks start without image waits, clusters, or
  optional/Konflux labels, and satisfy the repository's required checks.
- Enforcement label addition/removal and unrelated label events, including
  their effect on active and pending runs.
- Planner failure: selected test jobs still start despite the failed
  dependency; unrelated label events still leave test work disabled.
- Setup and test failures: later ordinary steps stop, while the intended
  diagnostics, reports, and Jira reporting still run.
- Cancellation: ordinary work, diagnostics, and Jira stop; report steps
  retain their `always()` behavior.

Reusable-output propagation, post-job hooks, downstream jobs, and check
aggregation also require live validation. Automatic execution of workflow
condition scenarios is a possible future follow-up if a suitable Go-based
approach becomes available. It is deferred from this change.
