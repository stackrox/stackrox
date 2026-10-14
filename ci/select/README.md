# CI test selection prototype

The Go selector decides which pull-request jobs to run. GitHub Actions and
Prow dispatchers apply that decision. The label `ci-dispatcher-enforce`
enables selection; without it, the gates run jobs and report the plan.

This is a port of the implementation at StackRox PR #23166, commit
`5457183f73f67bfe6f7e13d591916b8ea1703a46`. It keeps that implementation's
sample rules, defaults, and enforcement behavior. The PR description and
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
the TOML, error-wrapping, and test dependencies. Python, `jq`, and `curl` are
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

The default configuration paths are `ci/test-domains.toml` and
`ci/decision-defaults`. Override them with `--rules` and `--defaults`.
Without `--decision-out`, the resolver writes the decision to stdout.

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

The `should-dispatch` job calls `ci-select gha-dispatch-ci`, replacing the
inline Python dispatcher. Go retrieves current PR labels and writes the
job map plus `wait-for-gha-images` and `wait-for-konflux-images` outputs.
It reads `GITHUB_EVENT_NAME`, the metadata variables listed earlier, and
`CI_IS_FORK`, `CI_IS_DRAFT`, and `CI_UI_ONLY`. Fork rejection is preserved.
With enforcement disabled, non-PR events, or unusable defaults, it follows
the existing dispatcher rules. If live label lookup fails, it logs a warning,
uses `PR_LABELS` from the GitHub event, and disables enforcement for that run.
Event labels may be stale; explicit job labels and the existing fork, draft,
UI-only, and Konflux rules still apply. With no event labels, it applies those
rules with an empty label list.

`prow-gate.sh` defines `ci_decision_allows`. It pipes the existing CI
library's `get_pr_details` JSON into `ci-select prow-ci JOB`. Go reads the
Prow environment, fetches comments, handles rehearsal labels in the PR
body, and resolves the decision. Only exit code 3 means skip; success and
errors let the job run. The wrapper does not build the binary, parse JSON,
fetch comments, or implement selection policy.

Prow uses `PULL_BASE_SHA` for the diff against `HEAD` and `PULL_PULL_SHA` for
the head commit timestamp. The synthetic merge timestamp is not used for
`/test` freshness. The GitHub client uses `GITHUB_TOKEN` or `GH_TOKEN`, and
`GITHUB_API_URL` when set. Requests have a 30-second timeout, up to three
attempts for transport/server failures, and pagination for labels/comments.

## Bootstrap and prototype limits

GHA uses `.github/actions/ci-select` to install the Go version from `go.mod`
and build the binary before the gate. Individual gated jobs tolerate build
failure and proceed with their normal work. The decision and e2e dispatch
workflows require a successful build. Prow builds once at dispatch entry;
build failure bypasses selection and runs the job.

### Open gap: build and runtime failure recovery

GHA's `should-dispatch` has no fallback if the selector cannot compile, is
missing, or fails to run. These failures block E2E dispatch. Defaults and the
label-lookup fallback work only while the binary is running.

The distribution model remains an open decision: build the selector in each
job, or provide a prebuilt binary. A prebuilt binary avoids compilation
failures in the dispatch job but can still fail to start or execute. Either
model needs a backup dispatch path independent of the selector binary,
including when both compilation and execution are unavailable. The fallback
policy must account for fork rejection, draft and label restrictions, and
image-wait outputs. This prototype leaves that recovery path unimplemented.

These integration costs still need validation in real CI:

- Cold jobs may download a Go toolchain and module dependencies before they
  can skip. The prototype uses the root module and adds no binary-distribution
  or cross-workflow artifact system.
- Prow images need a usable Go compiler and access to required downloads.
  A build failure is visible in logs but means that job will not be skipped.
- GHA's `should-dispatch` uses `ubuntu-24.04` instead of `ubuntu-slim` to
  support the shared toolchain setup.
- Skipped GHA jobs still allocate runners and check out the repository.
  Existing outer workflow conditions remain in effect. Prow setup performed
  before `dispatch.sh` remains outside this gate.
- The rules and job inventory are samples, not a reviewed production map.
  In particular, the sample OCP target name and the dispatched Konflux target
  name differ; an unenrolled target follows existing dispatch rules.
- `image-wait` still requests image waiting on every PR. A language port
  does not make those waits cheaper or change required-check behavior.

The Go tests retain the 18 resolver fixture cases and 33 invalid-rule cases
from the Python prototype. They compare decisions and explanation logs
(excluding the timestamp) and the complete selected-job set. The fixture
loaders check counts and unique names so a case cannot disappear silently.
Rejected-rule tests check the rejection reason; typed TOML decoding errors
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
