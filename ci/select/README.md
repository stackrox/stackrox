# CI test selection

This prototype decides which pull-request jobs to run. You can try that decision on your machine. Nothing here talks to GitHub, and nothing here starts a cluster.

Two programs do the work. The resolver decides. A dispatcher carries the decision out and does not decide again. There is one dispatcher for GitHub Actions and one for OpenShift CI (Prow).

You need Python 3.11 or newer. Run the commands from the repository root.

## What you have

The resolver reads three things:

- A text file of changed paths, one path on each line. An empty file means the pull request changed no files.
- A text file of labels, one label on each line. An empty file means there are no labels.
- The rules in `ci/test-domains.toml`, and the per-target defaults in `ci/decision-defaults`.

It writes two files. The decision file is the list of jobs to run, one name on each line. A line that starts with `#` is a comment. A comment does not share a line with a job name. The log is the story of how the rules were applied. The decision file does not contain that story.

A job that is not on the decision list is skipped. A job that is not listed in `ci/decision-defaults` is outside this tool. The dispatchers leave that job alone.

## Assumptions

These are the choices the sample is built on. Several of them are temporary.

- This tool decides for pull requests. A push to `master` or to a `release-*` branch keeps the triggers that already exist.
- Every enrolled target has its own default, and today every default is `skip`. That keeps a half-finished rule file from starting the whole suite. Change a line in `ci/decision-defaults` when you want a different fallback for one target.
- If one rule says run and another says skip, the job takes its line from `ci/decision-defaults`. With the current file, that job is skipped.
- The label `ci-run-all-tests` asks every target to run. A skip from another rule is still a clash, so that target still takes its default. On a docs-only change, `go` stays off the list even when the label is present.
- The last rule says `default` for every target that no earlier rule mentioned. Those targets also take `ci/decision-defaults`.
- An empty changed-files file means no files changed. A diff that could not be read is a failure. The dispatcher then uses `ci/decision-defaults` and does not treat the failure as an empty diff.
- An empty decision file is not a decision. A decision file that contains only comments runs no jobs.
- A job the sample does not list keeps running the way its workflow already runs. The enforce label does not affect it.
- The label `ci-dispatcher-enforce` is the switch. Without that label, GitHub Actions and Prow still start jobs the way they do today. The tools only print what they would have done.
- A skipped GitHub Actions job in this sample still starts and exits successfully, so a required check can merge. The hooked jobs are `style-check` and `go`.
- A `/test` comment can run a job the decision skipped. A comment cannot take a job off the list. The Prow dispatcher treats a comment as that request when the comment time is at or after the commit time. If the comment list cannot be read, the comment does not force the job.
- Prow does not tell the pod that a comment started it. The dispatcher reads pull-request comments instead.
- If `dispatch.sh` cannot run Python 3.11, the OpenShift CI job runs.
- The `go` job is one target. Both of its matrix legs follow that one decision. The check name `go (GOTAGS="")` is not a separate target in the sample rules.
- `image-wait` runs `wait-for-images` on every pull request. That job requires `should-dispatch`, so the resolver adds `should-dispatch` to the list.
- The UI-only exit already in `dispatch.sh` runs before this decision. A job the decision wanted to run can still exit there when the change is only under `ui/`.
- The sample rules are a way to try the mechanism. They are not the map of which product directory runs which suite.
- The style, unit-test, and end-to-end dispatch workflows also run when any label is added or removed, so that `ci-dispatcher-enforce` takes effect without a new commit.

## Decisions

The decision file is the contract. Both dispatchers read that list. They do not look at the rules again to change it. In the commands below, each dispatcher resolves from the same inputs so you can see `run` or `skip` without copying a file by hand. In a workflow, a failed resolver is replaced by `ci/decision-defaults`.

A target ends up in one of these states:

- One or more rules say run, and none say skip. The job runs.
- One or more rules say skip, and none say run. The job is skipped.
- Some rules say run and some say skip. The job takes its default.
- No earlier rule mentioned the job. The last rule says `default`, so the job takes its default.
- A job that is going to run requires another job. The resolver adds the required job and says so in the log, even when that job's own result was skip.

Rules are walked from top to bottom. The `when` field is one of `label-exists`, `no-file-changed`, `any-file-changed`, `all-files-changed`, `always`, or `remaining`. A rule lists jobs under `run`, `skip`, or `default`. The value `["*"]` means every target. The last rule must be `remaining`, and it must use `*` for exactly one of those three lists.

Paths are regular expressions. `all-files-changed` matches only when every changed path matches. `any-file-changed` matches when one path matches.

The sample rules do the following:

- `ci-run-all-tests` runs every target, subject to the clash rule above.
- `go.mod`, `go.sum`, `proto/`, or `generated/` runs every target.
- A change that is only docs runs `style-check` and skips `go`.
- Any `.go` file runs `go`.
- Any changed file runs `style-check`.
- A `sensor/` file runs `sensor-integration-tests` and skips `go-postgres`.
- A `central/policy/` file runs `go-postgres` and skips `sensor-integration-tests`.
- Every pull request runs `wait-for-images`, which pulls in `should-dispatch`.

## Run the resolver

Create two input files in a scratch directory. This docs-only case has no labels:

```bash
mkdir -p /tmp/ci-select
: > /tmp/ci-select/labels.txt
printf '%s\n' README.md > /tmp/ci-select/files.txt

python3 ci/select/resolver.py \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d \
  --decision-out /tmp/ci-select/decision.txt \
  --log-out /tmp/ci-select/resolver.log
```

Exit status 0 means both output files were written. The decision file is:

```text
# docs-only and style
style-check
# image-wait
wait-for-images
# prerequisite of wait-for-images
should-dispatch
```

`go` is absent, so a dispatcher skips it. The log starts like this. Later lines record the default `skip` for every target the rules did not name:

```text
# stackrox/stackrox PR 23035
a1b2c3d: rule "docs-only" runs target "style-check" because file "README.md" changed
a1b2c3d: rule "docs-only" skips target "go" because file "README.md" changed
a1b2c3d: rule "style" runs target "style-check" because file "README.md" changed
a1b2c3d: rule "image-wait" runs target "wait-for-images"
a1b2c3d: added target "should-dispatch" because target "wait-for-images" requires it
```

A sensor file and a Central policy file disagree. `go-postgres` and `sensor-integration-tests` clash. Both defaults are `skip`, so neither name is in the decision file. `go` is on the list because the paths end in `.go`.

```bash
printf '%s\n' sensor/common/foo.go central/policy/service.go > /tmp/ci-select/files.txt

python3 ci/select/resolver.py \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d \
  --decision-out /tmp/ci-select/decision.txt \
  --log-out /tmp/ci-select/resolver.log

grep clash /tmp/ci-select/resolver.log
```

```text
a1b2c3d: clash on target "go-postgres"; rule "central-policy" says run, rule "sensor" says skip; default "skip"
a1b2c3d: clash on target "sensor-integration-tests"; rule "sensor" says run, rule "central-policy" says skip; default "skip"
```

To see a clash resolve to run, change the `go-postgres` line in a copy of `ci/decision-defaults` from `skip` to `run` and pass that copy with `--defaults`. The checked-in file stays all `skip`.

The same docs-only files, plus the label, still leave `go` off the list:

```bash
printf '%s\n' README.md > /tmp/ci-select/files.txt
printf '%s\n' ci-run-all-tests > /tmp/ci-select/labels.txt

python3 ci/select/resolver.py \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d \
  --decision-out /tmp/ci-select/decision.txt \
  --log-out /tmp/ci-select/resolver.log

grep clash /tmp/ci-select/resolver.log
```

```text
a1b2c3d: clash on target "go"; rule "run-all-label" says run, rule "docs-only" says skip; default "skip"
```

If `ci/decision-defaults` is missing a target that the rules name, the resolver prints `resolver failed: ...` on standard error, exits 1, and writes neither file.

## Run the GitHub Actions dispatcher

Use the same labels file and files file as the resolver. Standard output is the single word `run` or `skip`. The log goes to standard error.

This asks what would happen to `style-check` for the docs-only case when the pull request has `ci-dispatcher-enforce`:

```bash
: > /tmp/ci-select/labels.txt
printf '%s\n' README.md > /tmp/ci-select/files.txt

python3 ci/select/gha.py gate \
  --job style-check \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d \
  --enforce
```

Standard output:

```text
run
```

The same inputs for `go`, without `--enforce`. The decision would skip `go`. The missing label means the job still runs:

```bash
python3 ci/select/gha.py gate \
  --job go \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d
```

Standard output:

```text
run
```

Standard error includes `go: decision would skip; label ci-dispatcher-enforce is absent so the job runs`.

Add `--enforce` and standard output becomes `skip`.

`print` writes the decision and the log to standard output. It does not answer for a single job:

```bash
python3 ci/select/gha.py print \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d
```

A job name that is not in `ci/decision-defaults` prints `run` even with `--enforce`.

## Run the Prow dispatcher

The Prow command answers for one OpenShift CI job. Standard output is again `run` or `skip`.

For the docs-only files, `gke-qa-e2e-tests` is enrolled and no rule runs it, so the enforced answer is skip:

```bash
: > /tmp/ci-select/labels.txt
printf '%s\n' README.md > /tmp/ci-select/files.txt

python3 ci/select/prow.py gate \
  --job gke-qa-e2e-tests \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d \
  --enforce
```

Standard output:

```text
skip
```

A `/test` comment at or after the commit forces `run`. The comments file is a JSON list of objects with `created_at` and `body`:

```bash
printf '%s\n' '[{"created_at":"2026-09-30T12:00:00Z","body":"/test gke-qa-e2e-tests"}]' \
  > /tmp/ci-select/comments.json

python3 ci/select/prow.py gate \
  --job gke-qa-e2e-tests \
  --rules ci/test-domains.toml \
  --defaults ci/decision-defaults \
  --labels-file /tmp/ci-select/labels.txt \
  --files-file /tmp/ci-select/files.txt \
  --comments-file /tmp/ci-select/comments.json \
  --commit-time 2026-09-30T11:00:00Z \
  --repo stackrox/stackrox \
  --pr 23035 \
  --commit a1b2c3d \
  --enforce
```

Standard output:

```text
run
```

`/test all` also forces the job. A comment whose time is before `--commit-time` does not. Omit `--enforce` and the answer is `run` either way.

In a real OpenShift CI pod, `.openshift-ci/dispatch.sh` calls `ci/select/prow-gate.sh`. You do not need that script for a local try. It needs pull-request metadata that exists only in CI.

## What a pull request does today

`.github/workflows/ci-decision.yaml` prints the plan on every pull request. It does not start or stop other jobs.

With the label `ci-dispatcher-enforce` on the pull request:

- `style-check` in `.github/workflows/style.yaml` follows the list. A skip exits successfully after checkout.
- The `go` job in `.github/workflows/unit-tests.yaml` does the same. Both matrix legs follow the one target named `go`.
- `e2e-dispatch.yaml` follows the list for the enrolled end-to-end jobs and for `wait-for-images`.

`dispatch.sh` does the same for an enrolled Prow job, and it exits before credentials and cluster setup when the answer is skip. The enrolled Prow names are `gke-nongroovy-e2e-tests`, `gke-qa-e2e-tests`, `gke-ui-e2e-tests`, and `ocp-vm-scanning-e2e-tests`.

Without the label, those jobs start as they do today.

## Run the tests

The tests use the same inputs as the commands above. They do not call GitHub and they do not start a job.

```bash
python3 -m unittest discover -s ci/select -p 'test_*.py'
```

## Work left before this can ship

The commands above are enough to see whether a rule does what you expect. Shipping this for every pull request still needs the following:

- Replace the sample rules with the real map from directories and labels to suites, including the build-and-test tags, the Go end-to-end tests, and the GKE or OpenShift jobs a subsystem change should run.
- Choose a real default for each target. ROX-36969 asks an unmatched file to run the full pipeline. The file in the tree skips every target instead.
- Decide whether `ci-run-all-tests` overrides a skip. Today it does not, because a skip is a clash and the clash takes the default.
- Teach the remaining pull-request jobs to read the decision. `go-postgres` and `sensor-integration-tests` are in the rules, and their workflows do not consult the list, so a skip in the decision does not skip those jobs. The same is true for the other jobs in `style.yaml` and `unit-tests.yaml`, and for the scanner, CRD, and compatibility workflows.
- Enroll the rest of the Prow jobs, or leave them out on purpose. A name missing from `ci/decision-defaults` always runs.
- Keep a required GitHub check from sitting in the expected state. Skipping `style-check` and `go` starts the job and exits 0. The other required checks are not on that path.
- Make a skip cheap. `image-wait` still waits for images on every pull request, and then the enrolled end-to-end jobs are skipped. Some OpenShift jobs create the cluster before `dispatch.sh`. Moving that creation next to the test is separate work, and it is what makes a Prow skip cheap. The UI-only exit in `dispatch.sh` can still drop a job the decision wanted to run.
- Set `always_run: true` in `openshift/release` for each enrolled Prow job, on each branch whose pull requests should use this tool. Jobs left at `always_run: false` stay outside it. That flag does not by itself change what runs after a commit lands on a branch.
- Replace the comment-time guess for `/test` with a signal that this pod was started by that comment, if Prow can provide one.
- Add a check that fails when the rule file names a missing directory, a missing suite tag, or a job that no longer exists.
- Watch a docs-only pull request and a subsystem pull request with `ci-dispatcher-enforce` before removing that label. Pushes to `master` and to release branches should keep running the full suite.
