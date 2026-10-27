# Docs-only product-test selection

Actions and enrolled Prow jobs skip product work when every relevant changed path
ends in `.md` or `.txt`, except these two inputs:

- `.openshift-ci/dev-requirements.txt` defines CI dependencies.
- `roxctl/common/report/testdata/two-images.txt` defines expected report test output.

Changing either excluded file retains normal product-test execution, including
when it changes alongside documentation. Mixed product changes and unavailable
data also retain existing execution. Builds, style checks, and selection
verification remain eligible. `ci-run-all-tests` has no selection meaning.

`file-groups.yaml` defines named path groups. Go evaluates `include` and optional
`exclude` expressions and emits one versioned `facts` object. An empty diff matches
no every-file group. Adding a group changes configuration, not workflow outputs.
All hand-edited selection configuration uses YAML. The loader rejects unknown
fields, duplicate keys, and additional documents. Filename inputs and decision
outputs use JSON.

Actions collects and validates the complete paginated PR diff, including both
rename paths. Jobs consume the facts through `detect-changes.yml`, retain their
matrices/check names, and guard setup, tests, uploads, coverage, and reporting.

Prow keeps its existing scheduling. `dispatch.sh` passes a normalized target to
the installed `ci-selection` binary. `prow-product-tests.yaml` enrolls exact test
targets. Unknown targets and builders retain execution. The binary uses immutable
Prow base/head revisions and compares the PR head with its merge base, with rename
detection disabled so both rename endpoints remain relevant. It never schedules
jobs. An explicit `/test` schedules a job but does not bypass this execution gate.

Intentional Prow skips carry shared state into `end.sh`. Cleanup still runs when
resources were created, and failures/cancellations remain visible. Runner/image
preparation and infrastructure provisioned before dispatch may still cost time.

## Draft demo against master

This candidate explicitly passes `--ignore ci/selection/demo-ignore.yaml` in both
adapters. That file lists only the implementation paths needed to exercise real
skips in a draft implementation PR against `master`. The facts include the ignored
paths so the filtered input is visible. The independent `CI selection verification`
workflow never uses the product selection result and tests the complete machinery.

This is a demo configuration, not a claim that CI changes are documentation.
**Before production merge, empty the demo exclusion array.** Verification fails
for a non-draft PR or a master push while exclusions remain enabled. Keep the regression
cases that verify full-diff production behavior. No temporary base branch is
needed for the demo.

For the first draft run, this PR's README change supplies a nonempty docs input
alongside the ignored implementation paths. Add a product-code change to the same
draft to demonstrate that relevant jobs retain normal eligibility. Remove that
product change with an ordinary follow-up commit to restore the docs-only demo.
Keep the label/event configuration constant between the runs. For GHA E2E jobs
that require labels on drafts, request the same existing suite label for both
cases. For Prow, request an existing job with its named `/test` command. Inspect
actual child steps and skip artifacts, not only successful wrapper checks.

If filtering removes every path, selection retains normal execution. API errors,
stale revisions, missing Git history, invalid configuration, missing binaries,
and invalid decision output cannot authorize a product skip.

## Local validation

From the repository root:

```sh
go test -race -cover ./ci/selection
go vet ./ci/selection
shellcheck -e SC1091 ci/selection/prow.sh .openshift-ci/dispatch.sh .openshift-ci/end.sh scripts/ci/jobs/test-binary-build-commands.sh
make github-actions-lint
```

Tests run the actual workflow collection and E2E dispatch scripts with fake API
responses. They also test rename endpoints, an advanced base branch, failure
fallbacks, product-step guards, Prow decisions, and cleanup outcomes. On macOS,
install a current Bash or set `NATIVE_TEST_BASH` to its path.
