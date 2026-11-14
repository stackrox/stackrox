# CI job selection

CI job selection decides which test suites a pull request can safely skip
based on the files it changes. A change to `README.md` does not need the
backend test suites; a change to a shared Go package does.

The rules live in [`ci/test-selection-rules.yaml`](../test-selection-rules.yaml).
The selector reads the changed files, applies the rules, and prints `run` or
`skip` for every enrolled job.

## The model is job-centric

Every job is decided by exactly one rule. Because a job belongs to one rule,
no two rules can disagree about it — there is nothing to resolve across rules,
and no cross-rule contradiction to detect. Validation only checks structure.

```yaml
version: 1

# The inventory of jobs these rules decide for. Each must be decided by
# exactly one rule below.
jobs: [go, go-postgres, sensor-integration-tests, e2e-ui-tests-gke, lint]

rules:
  # run_if_changed: default skip, run only when a changed file matches.
  - name: postgres
    jobs: [go-postgres]
    run_if_changed: ['^central/', '^pkg/']

  # skip_if_only_changed: default run, skip only when every changed file matches.
  - name: go-unit
    jobs: [go, sensor-integration-tests]
    skip_if_only_changed: ['^ui/', '^.*\.md$']

  - name: ui-e2e
    jobs: [e2e-ui-tests-gke]
    skip_if_only_changed: ['^.*\.md$']

  # always_run: run on every pull request, whatever the diff.
  - name: lint
    jobs: [lint]
    always_run: true
```

### Fields

| Field | Meaning |
| --- | --- |
| `version` | Rules format version. Only `1` is supported. |
| `jobs` | Inventory of job names the rules decide for. Listing a name does not create a workflow job. |
| `name` | Unique, nonempty rule name, used in the trace. |
| `jobs` (rule) | The jobs this rule decides. Each inventory job appears in exactly one rule. |
| `run_if_changed` | Positive patterns. Default skip; the rule's jobs run when any changed file matches. |
| `skip_if_only_changed` | Negative patterns. Default run; the rule's jobs skip when every changed file matches. |
| `always_run` | `true` to run the rule's jobs on every pull request. |

A rule must declare **exactly one** of `run_if_changed`, `skip_if_only_changed`,
or `always_run`. The one clause it declares sets the baseline, so there is no
cross-clause conflict to resolve. To carve an exception (run a subset of a skip,
or skip a subset of a run), use a `!` exclusion inside that clause — see below.

### Patterns

Patterns are rooted Go regular expressions matched against repository-relative
paths, anchored at the start of the path (they need not match the whole path).
Prefix an entry with `!` to exclude: a file it matches is treated as not
matching that pattern set. For example, `skip_if_only_changed` can treat `.txt`
files as documentation while excluding a prefetch list that tests depend on:

```yaml
skip_if_only_changed:
  - '^.*\.txt$'
  - '!^tests/images-to-prefetch\.txt$'
```

## Decision semantics

Each rule decides its jobs by its single clause:

| Clause | Decision for the rule's jobs |
| --- | --- |
| `run_if_changed` | run if any changed file matches it, else skip |
| `skip_if_only_changed` | skip if every changed file matches it, else run |
| `always_run` | always run |

An **empty diff** runs every job — nothing changed that a skip rule could clear.

### Exceptions with `!`

Because a rule has one clause, exceptions are expressed as `!` holes within it,
not a second clause. A `!` entry only ever forces a *run* (it removes a file
from the skip set, so the skip can no longer cover it):

```yaml
# Skip Sensor for operator-only diffs, but still run it when the bundled
# charts change (a subset of operator/).
skip_if_only_changed: ['^operator/', '!^operator/tests/chart/']
```

This covers the cases a combined run/skip rule would — see the git history for
why the two clauses are mutually exclusive.

## Enforcement

On GitHub pull requests, add the `ci-dispatcher-enforce` label to apply the
skip decisions. Without the label, the selector prints a preview and the
workflow keeps its existing behavior. Anything that goes wrong — bad rules, a
failed diff, missing output — fails open: every job runs.

The selector does not infer dependencies from the changed code. The path rules
record boundaries that have been reviewed; a new import can make a skip wrong
until the rules file is updated.

## Commands

```sh
ci/select/build.sh                               # build ci/select/bin/ci-select
ci/select/bin/ci-select validate                 # lint the rules file
ci/select/bin/ci-select resolve --files-file F   # decide for a newline-delimited file list
ci/select/bin/ci-select gha                      # resolve a PR from the GitHub event (CI)
```

## Layout

| Path | Role |
| --- | --- |
| `internal/rules` | Rule model, parsing, and path matching |
| `internal/resolver` | The decision function and its trace |
| `internal/validation` | Structural lint (`Load`) |
| `main.go` | `validate` / `resolve` / `gha` CLI |
| `diff.go` | Computes the changed-file list with git |
| `gha.go`, `gha-gate.sh`, `gha-consume.sh` | GitHub Actions integration |
