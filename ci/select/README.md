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

A job that is not on the decision list is skipped. A target that `ci/decision-defaults` does not name is skipped. An empty decision file is logged, and the dispatcher follows that defaults file instead.

## Assumptions

This prototype relies on the surrounding CI for the following:

- The rules are a TOML file, `ci/test-domains.toml`.
- GitHub Actions triggers the hooked workflows, so the dispatcher runs inside a job that has already started.
- Each enrolled Prow job has `always_run` set to `true`. Prow starts the job, and this tool can then skip the test. That flag lives in `openshift/release`, outside this repository.
- When one rule says run and another says skip, the target takes its line from `ci/decision-defaults`.
- The last rule says `default` for every target that no earlier rule mentioned. Those targets take `ci/decision-defaults`.
- Adding or removing any label reruns the style, unit-test, and end-to-end dispatch workflows, so `ci-dispatcher-enforce` takes effect without a new commit. Whether a label change is a legitimate trigger is open for review.

## Requirements

The tool must follow these rules:

- Decide for pull requests. A push to `master` or to a `release-*` branch can later use the same decision. Until that port, those pushes keep the triggers they have now.
- Let `ci/decision-defaults` record which checks run on a pull request today, and which do not. Derive each run and each skip from that file. A target the file does not name is skipped.
- When the label `ci-run-all-tests` is set, use a second default: the wider set of checks that label runs today. That set is not the pull-request default above.
- Treat an empty changed-files file as a pull request that changed no files. Treat a diff that could not be read as a failure, and then use `ci/decision-defaults`.
- When the decision file is empty, or contains only comments, log that and follow `ci/decision-defaults`. When a job has no line in that file, log that and skip the job.
- Use the label `ci-dispatcher-enforce` as the switch. Without that label, GitHub Actions and Prow still start jobs the way they do today. The tools only print what they would have done.
- When a hooked GitHub Actions job is skipped, start the job and exit successfully, so a required check can merge. The hooked jobs are `style-check` and `go`.
- Give an explicit `/test` priority over the decision. The Prow dispatcher runs that job when a comment at or after the pull-request head says `/test` and names the job or `all`, even when the decision skipped it. In an OpenShift CI pod that time comes from `PULL_PULL_SHA`. The synthetic merge commit is not that time. The comment does not remove a job the decision listed. The resolver does not issue these commands and does not use them when it builds the list. If the comment list cannot be read, the job runs, so a missing list cannot hide a command.
- Treat `go` as one target, so both matrix legs follow one decision. `go (GOTAGS="")` is only the GitHub check name.
- Run `wait-for-images` on every pull request, through the `image-wait` rule. That job requires `should-dispatch`, so the resolver adds `should-dispatch`. In production, `should-dispatch` is boilerplate: the resolver and the dispatchers replace it.
- Let the resolver decide every job, including a change that only touches `ui/`. There is no separate UI-only exit.
- Keep the checked-in rules as a sample for trying the mechanism. The map from each product directory to its suite comes later.

## Decisions

The decision file is the contract. Both dispatchers read that list. They do not look at the rules again to change it. In the commands below, each dispatcher resolves from the same inputs so you can see `run` or `skip` without copying a file by hand. In a workflow, a failed resolver is replaced by `ci/decision-defaults`.

A target ends up in one of these states. A clash takes the default, and the last rule says `default`. Both of those are assumptions, listed earlier.

- One or more rules say run, and none say skip. The job runs.
- One or more rules say skip, and none say run. The job is skipped.
- Some rules say run and some say skip. The job takes its default.
- No earlier rule mentioned the job. The last rule says `default`, so the job takes its default.
- A job that is going to run requires another job. The resolver adds the required job and says so in the log, even when that job's own result was skip.

Rules are walked from top to bottom. The `when` field is one of `label-exists`, `no-file-changed`, `any-file-changed`, `all-files-changed`, `always`, or `remaining`. A rule lists jobs under `run`, `skip`, or `default`. The value `["*"]` means every target. The last rule must be `remaining`, and it must use `*` for exactly one of those three lists.

Paths are regular expressions. `all-files-changed` matches only when every changed path matches. `any-file-changed` matches when one path matches.

The sample rules do the following:

- The checked-in `ci-run-all-tests` rule asks every target to run, and a skip from another rule still clashes. The requirement is a second default: the wider set of checks that label runs today.
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

The checked-in rule still asks every target to run. On these docs-only files the label clashes with the skip for `go`, so `go` stays off the list. The requirement is a separate, wider default for this label:

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

A job name that is not in `ci/decision-defaults` prints `run` even with `--enforce`. The requirement is to log that name and skip it.

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

An explicit `/test` for this job forces `run`, even with `--enforce`. The comments file is a JSON list of objects with `created_at` and `body`. A comment at or after `--commit-time` counts. `/test all` counts too.

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

In a real OpenShift CI pod, `.openshift-ci/dispatch.sh` calls `ci/select/prow-gate.sh`. You do not need that script for a local try. It needs pull-request metadata that exists only in CI. If that script cannot run Python, or cannot read the comment list, the job runs.

## What a pull request does today

`.github/workflows/ci-decision.yaml` prints the plan on every pull request. It does not start or stop other jobs.

With the label `ci-dispatcher-enforce` on the pull request:

- `style-check` in `.github/workflows/style.yaml` follows the list. A skip exits successfully after checkout.
- The `go` job in `.github/workflows/unit-tests.yaml` does the same. Both matrix legs follow the one target named `go`.
- `e2e-dispatch.yaml` follows the list for the enrolled end-to-end jobs and for `wait-for-images`.

`dispatch.sh` does the same for an enrolled Prow job, and it exits before credentials and cluster setup when the answer is skip. The enrolled Prow names are `gke-nongroovy-e2e-tests`, `gke-qa-e2e-tests`, `gke-ui-e2e-tests`, and `ocp-vm-scanning-e2e-tests`. Before that decision, `dispatch.sh` still has its own exit for a change that is only under `ui/`, for job names matching `nongroovy` or `upgrade`. The requirement is to drop that exit and let the resolver decide those jobs too.

Without the label, those jobs start as they do today. With the label, an explicit `/test` for this job still runs it when the decision skipped it. The gate reads pull-request comments to see that command. If it cannot read them, the job runs.

## Run the tests

The tests use the same inputs as the commands above. They do not call GitHub and they do not start a job.

```bash
python3 -m unittest discover -s ci/select -p 'test_*.py'
```

## Work left before this can ship

The commands above are enough to see whether a rule does what you expect. Shipping this for every pull request still needs the following:

- Replace the sample rules with the real map from directories and labels to suites, including the build-and-test tags, the Go end-to-end tests, and the GKE or OpenShift jobs a subsystem change should run.
- Fill `ci/decision-defaults` with the checks a pull request runs today, and add a second default for `ci-run-all-tests`: the wider set that label runs today. The file in the tree skips every target instead, and the checked-in label rule asks every target to run.
- Teach the remaining pull-request jobs to read the decision. `go-postgres` and `sensor-integration-tests` are in the rules, and their workflows do not consult the list, so a skip in the decision does not skip those jobs. The same is true for the other jobs in `style.yaml` and `unit-tests.yaml`, and for the scanner, CRD, and compatibility workflows.
- Log a job that `ci/decision-defaults` does not name, and skip it. The dispatcher still prints `run` for that name.
- Keep a required GitHub check from sitting in the expected state. Skipping `style-check` and `go` starts the job and exits 0. The other required checks are not on that path.
- Make a skip cheap. `image-wait` still waits for images on every pull request, and then the enrolled end-to-end jobs are skipped. Some OpenShift jobs create the cluster before `dispatch.sh`. Moving that creation next to the test is what makes a Prow skip cheap. `should-dispatch` becomes boilerplate once the resolver and the dispatchers replace it.
- Remove the UI-only exit in `dispatch.sh` once the rules cover a change that is only under `ui/`. Deleting it before that would stop the current skip for every such pull request, including ones without `ci-dispatcher-enforce`.
- Set `always_run: true` in `openshift/release` for each enrolled Prow job, on each branch whose pull requests should use this tool. Jobs left at `always_run: false` stay outside it. That flag does not by itself change what runs after a commit lands on a branch.
- Add a check that fails when the rule file names a missing directory, a missing suite tag, or a job that no longer exists.
- Watch a docs-only pull request and a subsystem pull request with `ci-dispatcher-enforce` before removing that label. A push to `master` or to a release branch can later use the same decision. Until then, those pushes keep the triggers they have now.
