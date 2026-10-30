# CI job selection

CI job selection decides which test suites a pull request can safely skip
based on the files it changes. For example, a change to `README.md` does not
need the backend test suites. A change to a shared Go package does.

The rules live in [`ci/test-selection-rules.yaml`](../test-selection-rules.yaml). The file lists
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

## Read a rules file

This illustrative example uses two targets and four rules:

```yaml
version: 1
jobs: [go, sensor-integration-tests]
rules:
  # Documentation outside test fixtures is safe without either test suite.
  - name: documentation
    when: any-file-matches
    paths: ['^docs/', '^[^/]+\.md$', '!^docs/testdata/']
    skip: [go, sensor-integration-tests]

  # Security documentation also requires Go tests.
  - name: security-documentation
    when: any-file-matches
    paths: ['^docs/security/']
    run: [go]

  # Central access-control changes require Go, but not Sensor tests.
  - name: central-access-control
    when: any-file-matches
    paths: ['^central/sac/']
    run: [go]
    skip: [sensor-integration-tests]

  # Run any job not already selected or skipped.
  - name: remaining
    when: remaining
    run: ['*']
```

The fields have these meanings:

| Field | Meaning |
| --- | --- |
| `version` | The rules format version. Currently only `1` is supported. |
| `jobs` | The inventory of target names that these rules can decide for. Listing a name does not create a workflow job. |
| `name` | A unique, nonempty rule name used in explanations. The name does not select the rule's behavior. |
| `when` | The condition under which a rule contributes. The supported values are `any-file-matches` and `remaining`, explained next. |
| `paths` | Rooted Go regular expressions matching repository-relative paths. Prefix an entry with `!` to exclude matches from this rule. Required for `any-file-matches`; omitted for `remaining`. |
| `skip` | Target names safe to skip for the listed paths. Optional for `any-file-matches`; omitted for `remaining`. |
| `run` | Target names required when any changed path matches this rule. `['*']` requests every enrolled target. The final `remaining` rule must use `run: ['*']`. |

### `when: any-file-matches`

An `any-file-matches` rule matches when at least one changed file matches
its `paths` patterns after exclusions.

### Path patterns

Patterns are regular expressions, not shell globs. After an optional `!`
prefix, each pattern must start with `^`, which anchors it to the repository
root. The root anchor applies to every alternative (`|`) in the pattern.
A path must match at least one inclusion and none of the exclusions:

- `'^docs/'` matches files anywhere under `docs/`.
- `'^[^/]+\.md$'` matches Markdown files in the repository root, such as
  `README.md`. It does not match `pkg/testdata/fixture.md`.
- `'^central/sac/'` matches files under that specific Central package.
- `'!^docs/testdata/'` removes files under `docs/testdata/` from this rule's
  coverage, even though `'^docs/'` also matches them.

Inclusion and exclusion order does not matter. Each rule must contain at
least one inclusion. Exclusions affect only their own rule: another rule
can still match the path and contribute its own run requests or skip
coverage. They do not force a global `run` decision. Quote `!` entries so
YAML treats them as strings.

The YAML examples use single quotes so `\.` reaches the regular expression
engine as a literal dot. Use `$` when a pattern must also match the end of
the path.

### `when: remaining`

The `remaining` rule says: **run every enrolled job not already explicitly
requested or skipped**. It must appear exactly once, as the last rule, and
must have `run: ['*']` with no `paths` or `skip` fields. Here `*` means all
still-undecided targets in `jobs`.

`any-file-matches` and `remaining` are the only `when` values supported today.

The final rule cannot select `skip`. Conditions based on labels, imports,
or dependencies are not part of this rules format. The GitHub enforcement
label controls whether to apply a computed decision; it is separate from
the rule language.

`remaining` defines normal selection behavior. Error recovery is separate:
invalid configuration, an unavailable planner, and requests for unknown
targets run tests even if the rules cannot be read.

## Composition of rules

The selector considers all matching path rules for each changed file. A rule's
`skip` list grants permission to skip those jobs for its matching files; its
`run` list requests those jobs whenever at least one file matches. A rule can
have both lists, but it must name at least one target and cannot put the same
target in both lists.

Decisions are made **file by file, then combined per job**. Think of each
changed file as carrying one card per job: every rule matching that file
stamps the card — either "run this job" or "safe to skip this job". One file
can carry both stamps.

For each job, the selector walks the files and combines the cards in this
order:

1. **Run requests survive only on files the job's skip rules do not cover.**
   On one card, if the file is both requested to run and marked safe to
   skip for that job, the run request is dropped **for that file only**.
   Documentation is the main case: a `README.md` inside a domain directory
   is covered by the documentation rule, so a domain rule's `run` request
   for that file does not count. A `.go` or `.proto` file in the same
   directory is not documentation, so its request stands.
2. If any run request survives on any file, run the job. One uncovered
   file with a surviving request is enough.
3. Otherwise, skip the job if the diff is nonempty and every changed file is
   covered by a rule that lists the job in `skip`. Different rules can cover
   different files, and multiple rules can contribute coverage for one file.
4. Otherwise, the final `remaining` rule runs the job. This includes an empty
   diff, which cannot provide skip coverage.

The selector considers all path rules, so their order does not affect the
result; the same file never depends on which rule was written first. An
exclusion removes a file only from that rule; other rules can still request
jobs or provide skip coverage for it. A `run` decision enables the suite
subject to its usual workflow eligibility.

Every decision is explained in the planner log and step summary. The trace
opens with a short legend and numbered, sectioned lines: each changed path
lists the rules it matched, then each numbered decision block states the
verdict first, with the rule that decided it — a job runs because a run
request survived on a path not marked safe to skip, runs because not every
path is covered so `"remaining"` runs it, or skips because every changed
path is covered and no run request survived. Evidence bullets under each
verdict list the run requests and their changed-path numbers, conflicts
where a skip overrode a run request (naming both rules and the file), the
covering rules, and the uncovered paths. Jobs with an identical verdict and evidence
print as one grouped block. A `-- decisions --` section separates the plain
per-target summary from the trace. See [EXAMPLES.md](EXAMPLES.md) for full
walkthroughs.

For the example rules, the results are:

| Changed files | Go | Sensor integration | Reason |
| --- | --- | --- | --- |
| `docs/guide.md` | skip | skip | The documentation rule covers the file for both jobs. |
| `docs/security/guide.md` | skip | skip | The documentation rule covers the file, so the security rule's Go request for it is dropped. Under the production rules — where Markdown in *any* directory is documentation — domain READMEs skip the same way. |
| `central/sac/example.go` | run | skip | The Central rule's Go request survives on the code file (the documentation rule does not match `.go`); the rule covers the file for Sensor integration. |
| `central/sac/README.md` | run | skip | The example documentation rule matches only `^docs/` and root-level Markdown, so this file is not covered and the Central request survives. |
| `docs/guide.md` and `pkg/shared.go` | run | run | The shared file has no skip coverage, so `remaining` runs both jobs. |
| `docs/testdata/fixture.md` | run | run | The documentation rule excludes the file, so it is not covered; `remaining` runs both jobs. |
| No files | run | run | No files provide skip coverage. |

Each `skip` entry must explicitly name a target in `jobs`; wildcard skips
are rejected. A new target has no skip coverage until a rule names it in
`skip`. In a path rule, `run: ['*']` requests every enrolled target when a
file matches. It must be the only entry in `run` and cannot accompany `skip`.

## Understand the current rules

The rules file has seven `any-file-matches` rules and one final `remaining` rule.
The table shows representative diffs and their results:

| Changed paths | Jobs skipped |
| --- | --- |
| UI files, optionally with eligible Markdown/text documentation | Everything except the two UI e2e suites, which `remaining` runs |
| Only `.md` and `.txt` files, outside the documented test/build exclusions and Central and Sensor domain rules | All fifteen targets |
| Documentation under a listed Central or Sensor domain | All targets except the jobs explicitly requested by those rules |
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

The Central and Sensor domain rules explicitly request Go tests and Sensor
integration tests, respectively. These requests apply only to files the
job's skip rules do not already cover: a domain README is covered by the
documentation rule, so its domain run request does not count, while `.go`,
`.proto`, and testdata files in the same directories still request their
suites.

The two frontend rules split one boundary by consumer: UI files contain
no Go code (`frontend-backend-only` skips the four Go and Sensor targets)
and the backend e2e suites drive APIs and clusters rather than the browser
(`frontend-no-backend-e2e` skips the nine non-UI e2e suites). The two UI
e2e suites stay uncovered on purpose: `remaining` runs them for any UI
change, because those suites exercise the built frontend.

Shared Go packages, generated code, protobufs, build scripts, and workflow
changes have no skip coverage and run every target. Operator API types
are shared inputs, so changes to them also run every target. New Operator
subdirectories need review before they join the rule's explicit path list.

The `documentation-only` rule matches `.md` and `.txt` extensions in any
directory, ignoring case, except for test/build inputs listed in its
`paths` exclusions. For example, `README.md` plus `sensor/notes.TXT` skips
all fifteen targets: the four backend targets and eleven e2e targets, provided
no other matching rule requests a run.
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
merge commit, builds the
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
and summaries preview the rule decisions. On the next run, removing the label
restores existing behavior, including the UI-only outer filters and the
release-variant label. With enforcement, the benchmark and Sensor outer filters admit the
job so selection owns the decision. Pushes and tags retain their existing
behavior.

### Temporary limitation: label changes require a new commit

Unit Tests, e2e Dispatch, and CI selector tests do not subscribe to `labeled`
or `unlabeled` events. Adding or removing `ci-dispatcher-enforce` does not start
these workflows, cancel an active run, or invalidate their check results. The
setting takes effect when a subsequent normal PR event starts a new workflow run.

After changing the label, push a new commit to refresh selection. If there
are no code changes, create and push an empty commit on the PR branch:

```sh
git commit --allow-empty -m "ci: refresh test selection"
git push
```

Pushing an unchanged branch does not trigger CI. Removing enforcement also
requires a new pushed commit before full test results replace earlier
selection-based skips. This is a temporary limitation of label-driven
configuration; these workflows create no runs for label changes.

### E2e jobs and required checks

The e2e dispatcher runs [`e2e_dispatch.py`](e2e_dispatch.py) after checkout.
The script validates the applied map and forwards one `ci-action`
(`run` or `skip`) through each variant's existing reusable workflows. Each
suite job starts with a small Bash decision step. A selected skip disables
checkout, setup, provisioning, tests, cleanup, and reporting while the job
finishes successfully. Existing check names and workflow nesting stay the
same. The PostgreSQL upgrade job also starts for a selected skip, even
though its version-check step did not run. The decision steps run before
their job's checkout, so they cannot call a script from the repository; the
integration test `TestSuiteDecisionStepsShareBody` pins them to one shared
body so copies cannot drift apart.

For enforced skip decisions, the dispatcher leaves each suite's normal
eligibility in charge: eligible suites start and skip their steps, while
ineligible variants (draft, optional-label, Konflux) do not start at all
and their checks finish as GitHub-native skipped conclusions, which also
satisfy required checks. It bypasses unneeded image-wait
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
checks out only the merge commit for the tree and fetches the two diff
endpoints with deepening shallow fetches until their merge base resolves;
unrelated history is never fetched, and a repository that already contains
the merge base fetches nothing. SHA fetches need no credentials because the
repository is public; if the remote rejects them, selection fails and every
test runs. Test jobs keep their existing checkout revision and need no extra
history for selection.

Consumers run on unreadable diffs, missing refs, invalid rules, failed
builds, missing or crashing executables, and unknown targets. Error recovery
runs independently of the normal `remaining` rule and requires no readable
target inventory. Configuration validation still fails on malformed input.

The planner builds before dependency preparation, PostgreSQL, image pulls,
or Kind. A failed build cannot execute a stale binary. The shell gate
recovers planner execution failures with `decisions={}`. Consumers parse
that output with `jq` in Unit Tests and the existing Python dispatcher in
e2e Dispatch, accepting only one JSON object with `run`/`skip`
values. Only an explicit `skip` for the consumer from a successful planner
suppresses work. Missing or malformed maps and planner failure run tests.
Explicit job status conditions bypass GitHub's default dependency-success
check so planner failure does not suppress eligible suites. Legacy
benchmark/Sensor UI-only filters apply when planning succeeds without
enforcement; planner failure admits these tests as well.
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

The Go unit tests cover the selector and use Git for temporary repository
fixtures. Go script integration tests live in `scripts_integration_test.go`,
behind the `integration` build tag. They execute the Bash wrappers with local
fixtures and the existing Python E2E dispatcher with a local `gh` stub.
Integration tests require Bash, `jq`, and Python 3 and make no API calls.

From the repository root:

```sh
ci/select/build.sh
ci/select/bin/ci-select validate
printf '%s\n' ui/example.tsx README.md > /tmp/ci-selection-files
ci/select/bin/ci-select resolve --files-file /tmp/ci-selection-files
go test -count=1 -race ./ci/select
go test -count=1 -race -tags=integration ./ci/select -run '^TestScript'
go vet ./ci/select
.gotools/bin/golangci-lint run --enable=unused ./ci/select
.gotools/bin/actionlint .github/workflows/unit-tests.yaml .github/workflows/ci-select-tests.yaml .github/workflows/ci-selection.yaml
.gotools/bin/actionlint -config-file .github/actionlint.yaml .github/workflows/e2e-*.yaml
shellcheck --norc -P SCRIPTDIR -x ci/select/build.sh ci/select/gha-gate.sh ci/select/gha-consume.sh
```

Go unit tests cover configuration validation, rule decisions, the CLI, Git
diff handling, and GitHub event/output handling. Script integration tests
cover planner failure recovery, decision parsing, and the dispatcher's
application of selection to check admission and image waits. They compare
fallback eligibility with normal dispatch behavior without fixing the whole
job inventory. A contract check compares the E2E targets in the policy,
dispatcher output, and reusable workflow calls. It reads workflow YAML without
requiring particular step names, ordering, or expression text. `CI selector tests`
runs both suites independently of suite selection; the existing `make github-actions-lint`
style check validates workflow syntax and expressions with `actionlint`.

Planning once per caller still requires a build and may download a Go
toolchain. There is no cross-run artifact or decision cache. Skipping these
jobs does not guarantee a short whole-PR turnaround while other workflows
run.

### Validation limits and possible follow-up

The Go integration tests execute the dispatcher and shell wrappers directly. They
verify missing/malformed decisions and planner failure recovery. `actionlint`
checks workflow syntax and expressions, but does not prove that decisions
reach every suite or that GitHub starts consumers after a failed or skipped
`needs` dependency. The small E2E gates that must run before checkout remain
inline. Their wiring and GitHub's scheduler behavior require live validation.

Before rolling out enforcement, inspect live Actions runs for these cases:

- A docs-only skip, a partial selection, and an uncovered change, including
  both Go matrix variants and successful checks for all eleven e2e variants.
  Confirm that docs-only e2e checks start without image waits, clusters, or
  optional/Konflux labels, and satisfy the repository's required checks.
- Label addition/removal: no new workflows start and active runs remain
  unchanged. A subsequent pushed empty commit refreshes the applied decision.
- Planner failure: selected test jobs still start despite the failed
  dependency.
- Setup and test failures: later ordinary steps stop, while the intended
  diagnostics, reports, and Jira reporting still run.
- Cancellation: ordinary work, diagnostics, and Jira stop; report steps
  retain their `always()` behavior.

Reusable-output propagation, post-job hooks, downstream jobs, and aggregation
of required checks also require live validation. Automatic execution of workflow
condition scenarios is a possible future follow-up if a suitable Go-based
approach becomes available. It is deferred from this change.
