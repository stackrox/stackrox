# Disposable label/check aggregation probe

Do not merge this branch. It intentionally fails required checks and removes
test execution from the two affected workflows. The probe branch starts at
core commit `3d43723521c41e5382ebc5ce53e8ec9f8b46f8c9` from PR #23716.
The disposable PR targets `master`: the core branch has no required-check rules.

## Probe behavior

The Unit Tests and e2e-dispatch workflows keep their event subscriptions,
job identities, job-level conditions, matrices, concurrency groups, selection
planner, change detector, and reusable-workflow call chains. Their runners
execute small probes instead of suites, image waits, or cluster operations.
Container setup and test-reporting outputs are omitted. E2E dispatch treats
the draft as ready for testing so the ordinary required E2E checks start.
The PostgreSQL-upgrade eligibility stub returns true so that check also starts.

| Ordinary check | Probe outcome |
| --- | --- |
| `go (GOTAGS="")` | Intentional failure |
| `go-postgres (GOTAGS="", 16)` | Wait ten minutes, then fail |
| `e2e-nongroovy-tests-gke / run / gke-nongroovy-e2e-tests` | Intentional failure |
| `e2e-gke-upgrade-tests / gke-upgrade-tests-sensor` | Wait ten minutes, then fail |
| Other eligible checks in these workflows | Success |

The probe does not introduce a check-name fix. Planner and detector behavior
is real. Existing unrelated workflows can still run normally on the PR.

## Run the experiment

1. Open the disposable PR as a draft against `master`, with `ai-assisted`.
2. Wait for the two intentional failures and for PostgreSQL and Sensor upgrade
   probes to be in progress. Collect a baseline snapshot with the command below.
3. Remove the `ai-assisted` label, then wait for the label-event workflow runs
   and their evidence collector to finish. Capture the merge box, including the
   individual required-check rows, while the original held checks are pending.
4. Add `ai-assisted` back and inspect the next label-event results. Do not push
   between these steps: the comparison must use the same PR head and merge SHA.
5. Let the held checks fail, then repeat the label removal/addition to observe
   aggregation when every probe has finished. Leave `ai-assisted` applied.
6. If more time is needed, rerun the ordinary workflow run that has action
   `opened` or `synchronize`, rather than an ignored-label run. Verify its probe
   steps have started before changing labels.

The collector reads APIs and writes local JSON only; it does not change labels,
trigger runs, edit rules, or merge anything. Label changes are separate actions.

```sh
python3 ci/label-check-probe/collect.py \
  --repository stackrox/stackrox --pr PROBE_PR_NUMBER \
  --intervals 0 --output /tmp/ci-label-probe-baseline
```

## Evidence

Each `CI label probe evidence` run saves snapshots at offsets of 0, 20, and
60 seconds and uploads a seven-day artifact. Snapshots contain:

- PR metadata and merge-state/status-check rollup.
- Active rules and the legacy branch-protection response, including API errors.
- Every available check run and commit status for both head and merge SHAs.
- Relevant workflow runs and job names, conclusions, steps, and attempts.
- Duplicate check names grouped by SHA, without assuming how GitHub aggregates them.

The collector records the event action, changed label, SHAs, and run identity.
Probe steps also print event metadata to their summaries. API permission errors
are preserved in JSON; a local collection with the authenticated `gh` CLI can
fill gaps in the workflow token's access.

Compare the exact required contexts and GitHub App IDs with check runs before
and after the unrelated label event. Inspect whether skipped matrix jobs retain
matrix suffixes, whether skipped reusable callers publish their parent name or
their nested names, and whether existing prefixed jobs remain isolated.
An overall blocked merge state alone is inconclusive because other checks and
review requirements can also block this disposable PR.

Only after establishing those results should a candidate naming fix be applied
and the same experiment repeated on its new SHA. Close the disposable PR after
the evidence has been downloaded; never merge its probe stubs.
