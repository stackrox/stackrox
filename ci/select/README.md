# CI job selection

CI job selection decides which test suites a pull request can safely skip
based on the files it changes. A change to `README.md` does not need the
backend test suites; a change to a shared Go package does.

The rules live in [`ci/test-selection-rules.yaml`](../test-selection-rules.yaml).
The selector reads the changed files, applies the rules, and prints `run` or
`skip` for every enrolled job.

## The model is job-centric

Every job is decided in exactly one place: the `always_run` list, or exactly
one rule. Because a job belongs to one rule, no two rules can disagree about
it — there is nothing to resolve across rules, and no cross-rule contradiction
to detect. Validation only checks structure.

```yaml
version: 1

# The inventory of jobs these rules decide for. Each must be decided exactly
# once below.
jobs: [go, go-postgres, sensor-integration-tests, e2e-ui-tests-gke]

# Jobs that run on every pull request, whatever the diff.
always_run: []

rules:
  # run_if_changed: run these jobs only when a changed file matches.
  - name: postgres
    jobs: [go-postgres]
    run_if_changed: ['^central/', '^pkg/']

  # skip_if_only_changed: skip these jobs only when every changed file matches.
  - name: go-unit
    jobs: [go]
    skip_if_only_changed: ['^ui/', '^.*\.md$']

  # A rule may use both. See the decision table below.
  - name: sensor
    jobs: [sensor-integration-tests]
    run_if_changed: ['^sensor/critical/']
    skip_if_only_changed: ['^sensor/']

  - name: ui-e2e
    jobs: [e2e-ui-tests-gke]
    skip_if_only_changed: ['^.*\.md$']
```

### Fields

| Field | Meaning |
| --- | --- |
| `version` | Rules format version. Only `1` is supported. |
| `jobs` | Inventory of job names the rules decide for. Listing a name does not create a workflow job. |
| `always_run` | Jobs that run on every pull request. |
| `name` | Unique, nonempty rule name, used in the trace. |
| `jobs` (rule) | The jobs this rule decides. Each inventory job appears in exactly one rule or in `always_run`. |
| `run_if_changed` | Positive patterns. The rule's jobs run when any changed file matches. |
| `skip_if_only_changed` | Negative patterns. The rule's jobs skip when every changed file matches. |

A rule must declare at least one of `run_if_changed` / `skip_if_only_changed`.

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

For each rule, given the changed files:

- `run_if_changed` **matches** when any changed file matches it.
- `skip_if_only_changed` **covers** when every changed file matches it.

| Clauses declared | Decision for the rule's jobs |
| --- | --- |
| only `run_if_changed` | run if it matches, else skip |
| only `skip_if_only_changed` | skip if it covers the diff, else run |
| both | run on a match, or on any file the skip set does not cover; otherwise skip. **Run wins** when both fire. |

An **empty diff** runs every job — nothing changed that a rule could clear.

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
