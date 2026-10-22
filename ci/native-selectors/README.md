# Native CI selector prototype

This discussion prototype evaluates native GitHub Actions conditions and Prow file selectors against PR #23403 at `34873a53e3b60b5fc7b50b1446b02d11a1eb0fca`. Read the [investigation](investigation.md) for the recommendation, behavior comparison, evidence, and known gaps. This branch is not intended to merge.

The Actions workflows are wired to the prototype. The Prow files are patches for the separate `openshift/release` repository; they have not been deployed. No label helper posts comments or starts jobs.

## Run local validation

From the repository root, use Python 3.9 or newer, Node.js compatible with `ui/package.json`, npm, Bash, Git, and jq:

```sh
cd ci/native-selectors
npm ci --ignore-scripts
npm test
```

The tests evaluate the actual workflow expressions with GitHub's expression library and compare them with recorded baseline decisions. They also check consumer gates, required check names, image retention, label-refresh failure, and real build-matrix construction with fixture context functions. They do not run product tests, publish images, or contact GitHub.

From the repository root, run the repository's workflow linter:

```sh
cd tools/linters
go run actionlint.kjanat.dev/cmd/actionlint -config-file ../../.github/actionlint.yaml \
  ../../.github/workflows/build.yaml \
  ../../.github/workflows/unit-tests.yaml \
  ../../.github/workflows/style.yaml \
  ../../.github/workflows/check-crd-compatibility.yaml \
  ../../.github/workflows/scanner-db-integration-tests.yaml \
  ../../.github/workflows/e2e-dispatch.yaml \
  ../../.github/workflows/native-ci-selection.yaml \
  ../../.github/workflows/native-ci-tests.yaml
```

To regenerate the oracle from the pinned implementation, first ensure the commit is available locally. The recorder creates a temporary source archive, adds the test fixture, and executes the baseline's own decision functions. It does not create a Git worktree or change the current branch. The baseline module requires Go 1.26.7 and its module dependencies.

```sh
python3 ci/native-selectors/record_baseline.py --output /tmp/native-ci-expected.json
python3 - <<'PY'
import json
from pathlib import Path
saved = json.loads(Path('ci/native-selectors/testdata/expected.json').read_text())
recorded = json.loads(Path('/tmp/native-ci-expected.json').read_text())
assert saved == recorded
print('Baseline outcomes match.')
PY
```

Do not refresh the saved expectations merely to make a failed comparison pass. Explain whether the baseline, policy, or intended execution changed.

## Exercise the external Prow configuration

The patch generator requires the seven source files listed in `prow/selectors.json` from `openshift/release` commit `3960ba6dc20f554e15fcf839e562a6d28353a2f3`. It checks each content hash. A source snapshot is sufficient for local patch verification; the release repository's own generation and validation need its full checkout.

From this repository's root:

```sh
python3 -m venv /tmp/native-ci-venv
/tmp/native-ci-venv/bin/pip install -r ci/native-selectors/requirements-test.txt
```

Set `release_snapshot` to the pinned source snapshot directory. Generate either profile without changing that snapshot:

```sh
/tmp/native-ci-venv/bin/python ci/native-selectors/prow/generate_patch.py \
  --release-root "$release_snapshot" --output /tmp/native-prow-keep-triggers.patch
/tmp/native-ci-venv/bin/python ci/native-selectors/prow/generate_patch.py \
  --release-root "$release_snapshot" --schedule-selected \
  --output /tmp/native-prow-schedule-selected.patch
/tmp/native-ci-venv/bin/python ci/native-selectors/prow/verify_release.py \
  --release-root "$release_snapshot"
```

`keep-triggers.patch` changes automatic GKE QA selection and leaves manual jobs manual. `schedule-selected.patch` also schedules the broad-file selection for QA/nongroovy variants and VM jobs. The latter changes scheduling semantics. Both preserve required UI execution and non-PR entries. Neither supports per-PR preview; deployment would enforce the chosen external configuration for applicable PRs.

To reproduce the upstream matcher experiment, obtain `pkg/config/jobs.go` from Prow commit `3f5cc7178d5a85a64fcf32c9cab401c937ee7218`. For example, this read-only command uses an authenticated GitHub CLI:

```sh
gh api 'repos/kubernetes-sigs/prow/contents/pkg/config/jobs.go?ref=3f5cc7178d5a85a64fcf32c9cab401c937ee7218' \
  -H 'Accept: application/vnd.github.raw+json' > /tmp/native-prow-jobs.go
python3 ci/native-selectors/prow/experiment.py /tmp/native-prow-jobs.go
```

The experiment verifies the source hash and compiles four unmodified matcher methods with minimal fixture types. Its 12 cases test path selection and forced triggering. Branch matching is fixed to `master`; webhook delivery, trust, reporting, and cluster execution are outside the experiment.

An actual external PR would apply one profile, run `make jobs`, `make checkconfig`, and `make -C ci-operator/config/stackrox/stackrox check`, then rehearse affected jobs. Those generation and live-integration checks were not run for this local prototype.

## Label-request helper

Save a PR webhook payload and a JSON object mapping current-head status contexts to their latest states. For example, a running UI check is represented as `{"ci/prow/gke-ui-e2e-tests":"pending"}`. The planner trusts that the caller supplied current-head statuses; it cannot verify this from that object alone.

```sh
python3 ci/native-selectors/prow/label_requests.py /tmp/pr-event.json \
  --statuses /tmp/pr-statuses.json
```

This prints `/test` commands for `area/ui` and `ci-run-all-tests`. Add `--schedule-selected` to request all 15 enrolled aliases for run-all. Without that option, run-all requests only the two normally automatic enrolled suites. `ci-all-qa-tests` does not request jobs. Pending and successful checks suppress duplicate requests.

Posting these commands requires a trusted bridge or a human. The bridge is not implemented. In particular, success alone cannot distinguish a completed test from a baseline gate's successful no-op. Prow's existing explicit `/test JOB` command is the reliable manual override; `/test all` still respects conditional selection.

## Runtime behavior and maintenance

The Actions helper collects path predicates from `paths.json`. Job policy lives in `.github/workflows/native-ci-selection.yaml`; E2E execution restrictions live in `e2e-dispatch.yaml`. Ordinary workflows preserve their outer conditions. Without `ci-dispatcher-enforce`, selection is preview-only. Workflows subscribe to label additions and removals.

Leave `CI_NATIVE_ALLOW_IMAGE_SKIPS` unset. Image builds remain enabled because Prow jobs, including the required UI job, still consume them. Enabling image skips requires an audit of all image consumers and their triggering behavior. It is not a step needed to try the local tests.

When changing policy, update the relevant expressions, path predicates, dependency expansion, and regression cases together. When changing Prow job configuration, update source hashes and aliases only after checking the new external revision. A shared build-time generator could reduce duplication, but this prototype does not introduce one.

The recorded results cover local decisions and configuration. Actual GitHub required-check reporting, draft/fork execution, label delivery, Prow comments, external job generation/rehearsal, and image availability still require CI access. No CI savings were measured.
