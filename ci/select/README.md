# ci-select

`ci-select` turns a CI selection rules file plus a pull request's changed
paths into a **plan**: a `run` or `skip` decision for every CI target. A
change to `README.md` does not need the backend test suites; a change to a
shared Go package does.

The rules live in [`ci/test-selection-rules.yaml`](../test-selection-rules.yaml).

Change detection is not done here: the changed paths are detected upstream
(the existing `.github/workflows/detect-changes.yml`, which uses the GitHub
API) and passed in with `--files-file`. The engine matches them against the
path rules.

## Structure

- **CLI** (`main.go`) — the `plan` and `validate` commands.
- **Engine interface** (`internal/ruleengine`) — the `Engine` interface and the
  `Decision` output type callers depend on. Swapping engines means adding an
  implementation behind this interface.
- **Engine** (`internal/ruleengine/jobcentric`) — the job-centric path-matching
  engine, built on `internal/rules` (parse + regexp matching),
  `internal/validation` (structure checks), and `internal/resolver` (run/skip
  decision + human-readable trace).

## The model is job-centric

Every job is decided by exactly one rule. Because a job belongs to one rule,
no two rules can disagree about it — there is nothing to resolve across rules,
and no cross-rule contradiction to detect. Validation only checks structure.

```yaml
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
| `jobs` | Inventory of job names the rules decide for. Listing a name does not create a workflow job. |
| `name` | Unique, nonempty rule name, used in the trace. |
| `jobs` (rule) | The jobs this rule decides. Each inventory job appears in exactly one rule. |
| `run_if_changed` | Positive patterns. Default skip; the rule's jobs run when any changed file matches. |
| `skip_if_only_changed` | Negative patterns. Default run; the rule's jobs skip when every changed file matches. |
| `always_run` | `true` to run the rule's jobs on every pull request. |

A rule must declare **exactly one** of `run_if_changed`, `skip_if_only_changed`,
or `always_run`. The one clause it declares sets the baseline, so there is no
cross-clause conflict to resolve. To carve an exception (run a subset of a skip),
use a `!` exclusion inside that clause — see below.

## How rules match and decide

A pattern is a Go regular expression, prefix-anchored to the repository-relative
path: it must match from the start but need not match the whole path, so
`^central/` matches every file under `central/`. Prefix an entry with `!` to make
it a *negative* pattern. A rule reaches its verdict in two steps.

**1. Per file.** A file **matches the set** when at least one positive pattern
matches it *and* no negative pattern matches it. A negative match always wins, so
`!` only ever removes a file from the set — it never adds one.

```yaml
skip_if_only_changed:
  - '^.*\.txt$'                        # positive: any .txt looks like docs
  - '!^tests/images-to-prefetch\.txt$' # except this one, which tests depend on
```

**2. Across the diff.** The clause folds that per-file result over every changed
file:

| Clause | Runs the rule's jobs when... |
| --- | --- |
| `run_if_changed` | **any** changed file matches the set (otherwise skip) |
| `skip_if_only_changed` | **not every** changed file matches the set — i.e. it skips only when all of them do (otherwise run) |
| `always_run` | always |

Because `!` only removes files from the set, it always pushes a rule toward
*run*, though by opposite mechanics:

- under `run_if_changed`, a vetoed file stops being a reason to run;
- under `skip_if_only_changed`, a vetoed file is no longer covered, so the
  "only changed" skip no longer holds and the jobs run.

An **empty diff** runs every job — nothing changed that a skip rule could
clear. A job with no recorded decision also runs, so the selector fails safe.

The selector does not infer dependencies from the changed code. The path rules
record boundaries that have been reviewed; a new import can make a skip wrong
until the rules file is updated.

## Build

```sh
make -C ci/select        # builds ci/select/bin/ci-select
```

## Usage

```sh
ci/select/bin/ci-select validate --rules ci/test-selection-rules.yaml

git diff --name-only origin/master...HEAD > changed.txt
ci/select/bin/ci-select plan --rules ci/test-selection-rules.yaml --files-file changed.txt
```

`validate` lints the rules file and reports every structural problem at once.
`plan` prints the run/skip decision per target; `--files-file` holds the
changed paths, one per line (omit it for an empty change set).

## GitHub Actions

`.github/workflows/ci-select.yaml` calls `detect-changes.yml` for the changed
paths, builds the binary, and prints the plan on every pull request. It is a
**preview only** — it gates no jobs. Wiring the plan into the test workflows is
a later step.
