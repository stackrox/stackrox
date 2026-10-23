# Job-local native CI selection

Each Actions job decides whether it needs to run. Its condition combines changed-file facts with PR labels. Prow makes automatic scheduling decisions through path selectors in `openshift/release`. No runtime program produces a list of jobs.

This is a discussion prototype. It exercises all 29 Actions job/helper targets and accounts for the four Prow suite groups in [PR #23403 at `34873a53e3b6`](https://github.com/stackrox/stackrox/commit/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca). Actions candidates and E2E dispatch match the saved comparison cases. **Prow does not have equivalent behavior.** Its gaps and concrete examples are in [cases.md](cases.md).

The [Granular Test Proposal](https://docs.google.com/document/d/1rpe0LZu39BLq9X_jUYH3iEWVmjqxkzIstQ8nwvMRv1I/edit) and [original CI design](https://docs.google.com/document/d/1qYJFtJr7_19X9nPliYgClDs-nNRRYA8FPe1bAg04jN0/edit) supply background proposals. Their unimplemented extensions are not attributed to the pinned comparison.

## Where to edit a rule

| Concern | Owning file |
| --- | --- |
| Go, PostgreSQL, Sensor integration, benchmarks, UI, shell, local roxctl | [unit-tests.yaml](../../.github/workflows/unit-tests.yaml), the job's `Explain selection` input |
| Style, generated files, Go/Actions/Prow lint, dependent image versions | [style.yaml](../../.github/workflows/style.yaml), the job's `Explain selection` input |
| Operator/CRD and Scanner database tests | [check-crd-compatibility.yaml](../../.github/workflows/check-crd-compatibility.yaml), [scanner-db-integration-tests.yaml](../../.github/workflows/scanner-db-integration-tests.yaml) |
| Eight Actions E2E consumers | [e2e-dispatch.yaml](../../.github/workflows/e2e-dispatch.yaml), each consumer's `should-run` condition |
| Image waits and their dependent jobs | The two waiter jobs in `e2e-dispatch.yaml`; image-build candidate in [build.yaml](../../.github/workflows/build.yaml) |
| Prototype regression tests | [native-ci-tests.yaml](../../.github/workflows/native-ci-tests.yaml) |
| Definitions of named filename groups | [file-groups.json](file-groups.json); these contain paths, never jobs or labels |
| Prow automatic scheduling | [prow/native-selection.patch](prow/native-selection.patch), an **unapplied** patch to external source configuration |

For example, PostgreSQL selects a candidate for a Central-only diff, a shared communication change, `area/central`, or a broad dependency/interface change. Its native expression contains those alternatives beside `go-postgres`. Adding an unrelated file can invalidate the Central-only alternative; a matching shared file or label can still select it.

## Runtime structure

```mermaid
flowchart LR
  API[PR filenames] --> D[Existing detect-changes workflow]
  D --> C[Go filename classifier]
  G[Path groups] --> C
  C --> F[Versioned file facts]
  F --> J[Conditions beside Actions jobs]
  L[PR labels] --> J
  J --> R[Existing tests and image waits]
  J --> S[Decision summaries]
  P[PR diff in Prow] --> N[Native release job selectors]
  N --> E[Existing Prow jobs]
```

The detector remains the only Actions file collector, once per calling workflow. Its existing `files` and `ui_only` outputs still support the existing outer conditions. They cannot express the comparison's other positive every-file predicates, such as Scanner-only or Central-only. [file-facts/main.go](file-facts/main.go) provides that missing classification using Go's standard library. There are no new Python or Bash programs or runtime dependencies.

Classification adds a checkout, Go setup and compilation to each calling workflow. This overhead is a cost of the prototype; no once-per-PR cache or measured runtime saving is claimed.

The classifier accepts one JSON object on stdin: `{"base":"…","head":"…","files":["ui/package.json"]}`. `--groups` defaults to `ci/native-selectors/file-groups.json`, a JSON object mapping group names to Go regular expressions. Patterns within a group combine with OR. Stdout is one JSON object: `{"version":1,"complete":true,"base":"…","head":"…","any":["ui"],"every":["ui"]}`. Empty diffs match no group. Invalid input or expressions cause a nonzero exit and no facts. The classifier has no Git/API access and knows nothing about labels, job names, dependencies, or dispatch.

The reusable detector exposes this as `facts`. API, checkout, compilation, or classification failure supplies `{"version":1,"complete":false,"any":[],"every":[]}`. It validates the event's head/base, paginates, and includes both rename paths. Multiline filenames, 3,000 or more collected paths, and oversized output use the unavailable-data fallback. The legacy `run_all` input can suppress `files`/`ui_only` without erasing known classification facts.

## How execution works

Without `ci-dispatcher-enforce`, Actions reports decisions and retains existing per-run test eligibility. With that label, native conditions apply selection. Ordinary jobs retain their check names and matrices; expensive steps become successful no-ops when unselected. Existing outer conditions still apply. Style remains selected. Jobs outside the comparison inventory retain their behavior.

E2E consumers keep their existing reusable workflows. Each consumer owns its native condition; `should-dispatch` only reads current labels and reports its own prerequisite candidacy. It exports no per-job plan. Consumers require a successful image waiter that permits execution. Waiters include the union of their consumers' rules. This duplication is necessary because Actions does not infer prerequisites from selected consumers.

Image builds **always remain enabled**. Their summary reports candidacy but does not enforce it: the comparison's dependency graph does not account for Prow and manual image consumers. Build metadata runs even if the added file-collection dependency fails, unless the workflow is cancelled. The execution scripts, build matrices, and child test workflows are unchanged.

`ci-run-all-tests` selects enrolled Actions candidates when classification succeeds. It does not bypass existing outer conditions, authorize forks, supply Konflux images, schedule Prow jobs, or choose the full QA test group. `ci-all-qa-tests` retains its separate role inside an already-running QA job. On classification failure, the comparison's defaults take precedence: the three Actions QA/FIPS E2E candidates skip; other enrolled Actions candidates run.

Enforced E2E selection can run a selected same-repository draft, as in #23403. Forks remain blocked. Enforced Konflux consumers require `konflux-build`, even with an explicit job label. Outside enforcement, legacy draft/label/image rules remain. Waiters now also reject forks explicitly, matching the comparison; the old dispatcher could wait for ordinary images on a ready fork without dispatching tests.

Label add/remove events reevaluate the selectable test workflows. This can restart work, including in shadow mode. Build keeps its original triggers because its images are retained; adding a label does not create another image build. Ordinary jobs use event labels, so rerunning an old event retains that event's labels. E2E refreshes current labels at the same head; refresh failure disables enforcement and uses event labels for legacy eligibility. [The cases](cases.md) explain this difference.

Decision summaries print already-evaluated candidates and permission to execute. The E2E report prints existing workflow results. Neither recomputes selection rules or claims tests ran from a successful wrapper/no-op check. There is no unified plan comment publisher.

## Validation

From the repository root, with Go 1.26.7, Git, jq, and Bash 4 or newer:

```sh
go -C ci/native-selectors/testing test -count=1 -v ./...
go -C ci/native-selectors/testing vet ./...
```

On macOS, set `NATIVE_TEST_BASH=/opt/homebrew/bin/bash` if necessary. Go tests build the classifier and use `act`'s Go expression evaluator against the actual workflow conditions. The three saved comparison fixtures remain unchanged: 2,175 PR candidate comparisons across 29 Actions targets and 83 E2E dispatch/waiter scenarios. Tests also check step gates, required-context structure, four actual build-matrix constructions, collector failures, label refresh, and selector-only external edits. Test dependencies live in a separate module; no verifier selects product jobs at runtime.

[The testing guide](testing/README.md) describes the test files and the known empty-label discrepancy between GitHub's expression library and `act`. It includes an expression that returns different results. The tests do not treat the evaluators as fully equivalent.

The optional focused Prow experiment uses the five unmodified matcher methods from the revision recorded in `prow/source.json`:

```sh
NATIVE_PROW_JOBS_GO=/path/to/pinned/prow/pkg/config/jobs.go go -C ci/native-selectors/testing test -count=1 -v ./...
```

Executed locally: those comparisons, 16 upstream matcher cases, focused repository-selected actionlint, Go build/vet and configured golangci-lint including unused checks, and whitespace checks. No CI time or cost savings were measured.

Still requiring CI access: real label deliveries/reruns; draft/fork approvals; exact matrix and reusable-workflow check reporting; image availability; Prow generated configuration checks/rehearsals; `/test`, `/test all`, and `/retest` routing; and actual test execution. Neither the external patch nor any workflow has been published or deployed by this work.
