# CI test selection prototype

The Go selector decides which pull-request jobs to run. GitHub Actions and
Prow dispatchers apply that decision. The label `ci-dispatcher-enforce`
enables selection; without it, the gates run jobs and report the plan.

This is a port of the implementation at StackRox PR #23166, commit
`5457183f73f67bfe6f7e13d591916b8ea1703a46`. It keeps that implementation's
resolver semantics and fixtures, with sample rules adapted for this PR
demonstration. The PR description and
some Python tests describe earlier policies; the implementation and the
fixture cases define this prototype's behavior.

## Build and test

Run from the repository root with the Go version in `go.mod`:

```sh
ci/select/build.sh
go test -race ./ci/select
ci/select/bin/ci-select validate
```

The build compiles only `ci/select`; it does not build Central, Sensor, or
roxctl. The executable is ignored by Git. The root module already provides
the YAML, error-wrapping, and test dependencies. Python, `jq`, and `curl` are
not required by the executable. Git is required to collect CI diffs and
Prow's head commit timestamp.

Run `ci/select/bin/ci-select` with no arguments for a command overview.
Every command accepts `-h` and `--help`, with flags and examples for that
command. For example, `ci/select/bin/ci-select help prow gate` explains the
local Prow gate, and `ci/select/bin/ci-select prow-ci -h` describes its CI
environment, stdin, and exit statuses.

## Try the resolver locally

Create a changed-path list and a label list, one item per line. For example,
use a file containing `README.md` and an empty labels file:

```sh
ci/select/bin/ci-select resolve \
  --files-file files.txt --labels-file labels.txt \
  --repo stackrox/stackrox --pr 23166 --commit a1b2c3d \
  --decision-out decision.txt --log-out resolver.log
```

The default configuration paths are `ci/test-domains.yaml` and
`ci/decision-defaults`. Override them with `--rules` and `--defaults`.
Without `--decision-out`, the resolver writes the decision to stdout.

The rules file uses YAML block lists and supports standalone and inline
`#` comments. Single-quote regular expressions to preserve backslashes, and
quote `'*'` so YAML treats it as a string rather than an alias.
Use one YAML document and omit optional fields instead of setting them to null.
Unknown fields are rejected. YAML values pass through JSON decoding so numbers
and booleans cannot silently become strings.

The decision lists jobs to run, one per line. An absent job is skipped.
`# no job runs` represents a valid decision to run nothing; blank text is
not a usable decision. Standalone comments are accepted, but inline
comments are rejected. The explanation log includes the commit, time,
matched rules, conflicts, defaults, and added dependencies.

## Rule behavior

The resolver evaluates rules in file order. Each matched rule gives its
named jobs a run, skip, or default vote. `*` names every job. A run vote
wins a conflict with a skip vote. A default is used only when there is no
run or skip vote. The final `remaining` rule covers jobs with no earlier
vote; it must use `*` for exactly one action.

For this PR demonstration, the checked-in `remaining` rule temporarily uses
the following policy:

```yaml
- name: remaining
  when: remaining
  skip:
    - '*'
```

Jobs run only when an earlier rule selects them or a running
job requires them. `ci-run-all-tests` still selects every job. The
`ci/decision-defaults` file remains the fallback for unreadable diffs or failed
resolution.

The supported conditions are:

- `any-file-matches`: at least one changed path matches at least one pattern.
- `every-file-matches`: every changed path matches a pattern, and the diff is nonempty.
- `no-file-changed`: the diff is empty.
- `label-exists`: the configured label is present.
- `always`: the rule applies regardless of files and labels.
- `remaining`: only the last rule may use it.

Patterns use Go's `regexp` syntax. The checked-in patterns are supported;
Python-specific lookarounds and backreferences are not supported. Invalid
patterns are rejected rather than translated.

A running job pulls in its `requires` dependencies, even if their own votes
said skip. Cycles, self-dependencies, unknown jobs, duplicate names,
overlapping actions within a rule, and mismatched defaults are rejected.
Job and explanation ordering follow the configuration rather than Go map
iteration order.

An empty diff is distinct from a diff that could not be read. The latter
uses the defaults. Dispatchers also use defaults when rules cannot be
loaded or resolved. If defaults are unusable, individual CI gates let the
job run. Jobs absent from the defaults are outside selection and run.

During fallback, `ci-run-all-tests` selects every target in the defaults.
A label matching a target's name also forces that target to run. Valid rules
still supply the dependencies of forced targets when the diff is unreadable.
If the rules cannot be loaded, the defaults must include required setup jobs.

The CI diff collector includes both sides of a rename by treating it as a
deletion and an addition. For example, moving `engine.go` to `README.md` still
selects Go tests because removing `engine.go` changes the compiled code.

## Try the dispatchers locally

The local dispatcher commands take the same file inputs as the resolver:

```sh
ci/select/bin/ci-select gha gate --job go \
  --files-file files.txt --labels-file labels.txt --enforce

ci/select/bin/ci-select gha print \
  --files-file files.txt --labels-file labels.txt

ci/select/bin/ci-select prow gate --job gke-qa-e2e-tests \
  --files-file files.txt --labels-file labels.txt --enforce \
  --comments-file comments.json --commit-time 2026-09-30T11:00:00Z
```

`gate` writes `run` or `skip` to stdout and explanations to stderr. Omit
`--enforce` to run the job while reporting the planned action. Use
`--files-unknown` instead of `--files-file` to exercise the defaults path.

The comments file is a JSON array of objects with `created_at` and `body`.
A `/test JOB` or `/test all` line at or after the PR head's committer time
forces that Prow job to run. Older or malformed timestamps do not force a
job. The CI adapter retrieves every comment page; an API failure lets the
job run so it cannot hide a `/test` request.

## CI entry points

`gha-gate.sh` calls `ci-select gha-ci`. Go collects the diff, parses labels,
resolves the decision, and writes `action=run` or `action=skip` to
`GITHUB_OUTPUT`. With no job argument it prints the plan and appends it to
`GITHUB_STEP_SUMMARY`. The wrapper's only fallback is `action=run` when the
binary is missing or fails.

The GHA gate reads `CI_BASE`, `CI_HEAD`, `CI_REPO`, `CI_PR`, `CI_COMMIT`,
`PR_LABELS` (comma-separated), and `CI_ENFORCE` (`true` enables selection).
`CI_DECISION_OUT` optionally saves the decision. `CI_SELECT_ROOT` locates
the checkout; the wrapper sets it automatically.

With no job argument, `CI_COMMENT_OUT` optionally saves a formatted PR comment.
The CI decision workflow creates or updates one `github-actions[bot]` comment
per PR, identified by `ci-resolver-decision`. It shows the short PR head SHA,
enforcement mode, and decision, with the resolver log collapsed under
**Resolver log**. Fork PRs retain the workflow log and step summary but do
not receive a comment. Runs for an outdated PR head do not post a comment.

The `should-dispatch` job calls `ci-select gha-dispatch-ci` directly.
Go reads PR labels from the event and writes the job map plus
`wait-for-gha-images` and `wait-for-konflux-images` outputs.
It reads `GITHUB_EVENT_NAME`, the metadata variables listed earlier, and
`CI_IS_FORK`, `CI_IS_DRAFT`, and `CI_UI_ONLY`. Fork rejection is preserved.
With enforcement disabled, non-PR events, or unusable defaults, it follows
the existing dispatcher rules. All GHA gates use the event's label snapshot;
dispatch makes no label API request. An unreadable diff or rules file uses
the defaults without disabling enforcement. A selected GHA e2e job always
enables its image-wait job, including during fallback. Fork restrictions and
the Konflux build-label requirement still apply.

When enforcement is enabled, enrolled GHA jobs bypass outer path filters and
let the selector decide. For example, `ci-run-all-tests` on a backend-only PR
can run UI tests. Preview mode retains the existing path filters.

GHA e2e job labels explicitly select their matching targets. Selected GHA
targets require `wait-for-images`, which requires `build-images`; Konflux
targets require `wait-for-konflux-images`. With enforcement enabled, no
selected GHA consumer means no image waiting. Image builds remain enabled on
every PR because Prow can run jobs outside the selector's inventory or accept
later `/test` requests. Konflux builds remain controlled by their existing
external integration and label.

The Build workflow resolves `build-images` once in `define-job-matrix` and
passes its action to every build job. The matrix stays populated so required
check names remain present. On a skip, every build, artifact, and push step
is skipped, and the jobs finish successfully. Image-wait jobs use the same
step-level approach, allowing downstream reusable workflows to report their
required e2e checks as skipped. No image placeholders are published.

`prow-gate.sh` defines `ci_decision_allows`. It pipes the existing CI
library's `get_pr_details` JSON into `ci-select prow-ci JOB`. Go reads the
Prow environment, fetches comments, handles rehearsal labels in the PR
body, and resolves the decision. Only exit code 3 means skip; success and
errors let the job run. The wrapper does not build the binary, parse JSON,
fetch comments, or implement selection policy.

Without enforcement, `dispatch.sh` skips nongroovy and upgrade jobs for UI-only
changes before building the selector. With enforcement, the selector controls
those skips.

Prow uses `PULL_BASE_SHA` for the diff against `HEAD` and `PULL_PULL_SHA` for
the head commit timestamp. The synthetic merge timestamp is not used for
`/test` freshness. The GitHub client uses `GITHUB_TOKEN` or `GH_TOKEN`, and
`GITHUB_API_URL` when set. Requests have a 30-second timeout, up to three
attempts for transport/server failures, and pagination for comments.

## Bootstrap and prototype limits

GHA uses `.github/actions/ci-select` to trust the checkout's Git ownership
before building the binary and collecting the diff. The action requires Go
1.21 or later on the runner or in the job container. It enables automatic
toolchain selection only for the build command, so Go can download the
toolchain required by `go.mod` without changing `PATH` or `GOTOOLCHAIN` for
later steps. Individual gated jobs tolerate build failure and proceed with
their normal work. The decision and e2e dispatch workflows require a
successful build. Prow builds once at dispatch entry; build failure bypasses
selection and runs the job.

CI integration will distribute prebuilt selector binaries in a separate change.
This prototype still builds the selector in each job. A build or execution
failure stops GHA e2e dispatch, including master and tag runs; this is an
acknowledged prototype limitation. Prebuilt binaries remove the compilation
dependency but can still fail to start or execute. Runtime failure recovery
remains an open design question.

These integration costs still need validation in real CI:

- Cold jobs may download a Go toolchain and module dependencies before they
  can skip. The prototype uses the root module and adds no binary-distribution
  or cross-workflow artifact system.
- Prow images need a usable Go compiler and access to required downloads.
  A build failure is visible in logs but means that job will not be skipped.
- GHA's `should-dispatch` uses `ubuntu-24.04` instead of `ubuntu-slim` to
  support the shared toolchain setup.
- Skipped GHA jobs still allocate runners and check out the repository.
  Preview mode retains existing outer path filters. Prow setup performed before
  `dispatch.sh` remains outside this gate.
- The rules and job inventory are samples, not a reviewed production map.
  All nine dispatched GHA e2e targets are enrolled. Konflux targets require
  the `konflux-build` label even when selection is enforced, because that
  label starts their PR image builds.
- Benchmarks and dependent collector/fact image checks are enrolled. Product
  Go changes select benchmarks; collector/fact version changes select their
  image checks. Miscellaneous policy checks always run.

## Remaining gaps

- **Selector delivery and execution failures (issue 3).** Prebuilt binaries are
  planned but are not delivered by this prototype. GHA e2e dispatch stops if
  the selector cannot build or execute. For example, a failed Go toolchain
  download on a master push prevents its e2e workflows from starting.
- **Unusable defaults (issue 2).** When the defaults file cannot be parsed,
  individual gates run their jobs and GHA e2e dispatch uses its existing
  eligibility rules. There is no shared emergency decision. For example,
  `go maybe` in the defaults lets the Go gate run, while a draft PR's automatic
  e2e jobs remain disabled. A common recovery policy still needs to be defined.
- **Unavailable dependency graph (issues 2 and 5).** If rules cannot be loaded,
  the resolver cannot recover dependencies from them. The checked-in defaults
  keep image prerequisites enabled, and GHA dispatch separately enables image
  waits for selected consumers. Other prerequisites need conservative defaults.
  For example, if a target `suite` requires `prepare`, both default to skip,
  and the rules file is invalid, a `suite` label runs `suite` without `prepare`.
- **Prow scheduling and command names (issues 1 and 5).** The selector gates
  Prow jobs after Prow schedules them. A run vote cannot start an opt-in job.
  For example, `ci-run-all-tests` does not schedule a Prow job configured with
  `always_run: false`; a Prow `/test` request is still needed. Variant names
  also need a mapping to internal target names. For example,
  `/test gke-oldest-ui-e2e-tests` schedules a variant that dispatches as
  `gke-ui-e2e-tests`, so the selector does not recognize that variant command
  as a force-run request.
- **Image-build savings (issue 1).** Image builds remain enabled on every PR
  until selection accounts for all Prow consumers and later manual requests.
  For example, a docs-only PR still builds images so a later `/test` request
  has images to use.
- **Production policy and live CI validation.** The rules remain samples, and
  unmatched targets are skipped by the demonstration catch-all. For example,
  a `Makefile`-only change selects style checks and image builds, but no test
  suites. Rule curation and live validation of required checks are still needed
  before broad enforcement. Local workflow tests do not prove that GitHub and
  Prow report every required status for a skipped run.

## Validation

The CI selector workflow validates the configuration on every PR before
selection can skip tests. A failed selector build fails this workflow.
For example, a misspelled `require:` key fails validation instead of silently
removing dependencies. The workflow must be made a required check separately
if configuration failures should block merging.

The Go tests retain the 18 resolver fixture cases and 33 invalid-rule cases
from the Python prototype. They compare decisions and explanation logs
(excluding the timestamp) and the complete selected-job set. The fixture
loaders check counts and unique names so a case cannot disappear silently.
Rejected-rule tests check the rejection reason; typed configuration errors
must identify the offending field.

Named resolver tests also check the design example, mixed CI/product changes,
unmatched paths, dependencies overriding skips, fallback defaults, invalid
configuration, and unusable defaults. CLI tests check empty diffs, both output
files, failure exit statuses, and the absence of output files on input errors,
including execution of the compiled binary.

The suite also covers current sample rules, dispatcher
fallbacks, fork rejection, `/test` timing, HTTP pagination, CLI outputs, git
diffs, and wrapper exit behavior. They do not contact GitHub or create CI
clusters.

The CI-tooling fixture records the complete changed-path list for this
prototype, including its build action and import-validator change. Tests
verify its enforced skips, shadow behavior, run-all override, and mixed
product changes.

`testdata/gha-required-checks.json` records the 22 required GitHub Actions
check contexts from the active `master` ruleset. Workflow tests verify that
their jobs remain defined and execute the default PR build-matrix script to
check its required variants. Other tests verify that build and image-wait
gates skip work at step level, preserving the checks and dependency chain.
Selector tests run when enrolled workflows or the build action change.

To demonstrate selection on the PR, compare runs triggered by removing and
adding `ci-dispatcher-enforce`, then add `ci-run-all-tests` to check the
override. With enforcement on, the CI-tooling diff skips PostgreSQL, Sensor,
scanner DB integration, benchmarks, image waiting, and e2e execution. Image
builds still run for possible Prow consumers. The gated jobs complete
successfully after skipping their work.
Verify the required contexts from the ruleset individually; a green workflow
alone does not prove that all required checks were reported. The required
Prow status and the PR's draft state remain outside this GHA demonstration.
