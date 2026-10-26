# Complex cases and native-selection boundaries

The reference is [PR #23403 at `34873a53e3b6`](https://github.com/stackrox/stackrox/commit/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca). “Candidate” means selected before event, trust, image, and existing job conditions. It does not mean tests ran. Future test-subgroup, platform, or architecture optimizations proposed in the source documents are not implemented by that comparison.

## Actions: rules and their interactions

The comparison combines positive votes, lets run win over skip, and finally skips undecided targets. For this fixed rule set, each job's condition can therefore OR its positive predicates, then account for prerequisites. Docs-only, CI-tooling-only, the subsystem skip lists, the four E2E skip lists, and `remaining` do not cancel a positive vote. They need no separate negative expression here. This simplification depends on that final skip rule; changing the policy requires reviewing the expressions.

| Rule family | Implementation and example |
| --- | --- |
| Always-run style | `style-check` has candidate `true`, including docs-only PRs. |
| Any-file rules | Go outside `ci/` and `tools/roxvet/` selects ACS Go and generated-file verification; product-root `.go` selects benchmarks; `.golangci.yml` selects Go lint; shell/scripts and workflow/Prow tooling select their own checks; version files select the dependent-image check. |
| Broad rules | `go.mod`, `go.sum`, `proto/`, `generated/`, or any `.proto` selects every enrolled candidate when facts are complete. Existing job restrictions can still block execution. |
| Positive every-file rules | UI-only selects UI/component; Scanner-only selects Go/Scanner DB; operator/config-controller/CRD-only selects Go/CRD checks; roxctl-only selects Go/local roxctl/Actions nongroovy. The classifier preserves the exact path groups. |
| Subsystem-only versus shared paths | Central-only selects PostgreSQL; Sensor-only selects Sensor integration. Shared libraries and Central/Sensor communication paths select both, even in mixed diffs. |
| All six area labels | Central, Sensor, Scanner, UI, operator, and roxctl labels select their respective candidates. The original outer conditions remain. |
| All nine E2E job labels | Each consumer's own expression includes its job label; waiters and the image-build candidate include dependent consumers. |
| Run-all, QA-group and enforcement labels | Run-all selects candidates; enforcement applies the decisions; `ci-all-qa-tests` changes the contents of an already-running QA workload. None grants trust or starts Prow jobs. |
| Failure defaults | Unknown facts select ordinary Actions targets and nongroovy/other non-QA E2E targets; the three Actions QA/FIPS targets remain unselected. Labels do not override these unknown-data defaults. |
| Non-PR events | Existing push, tag, reusable-call and manual triggers remain. PR enforcement is disabled. The existing matrix expansion and manual image-tag inputs remain. |
| Selector changes | The comparison's selector-tool paths select regression tests. Paths for this native implementation additionally select those tests. This is an explicit extension. |
| Explicit repository overrides | A validated match of a workflow's `run_all` patterns selects its candidates. Detector and unit-workflow edits select CI verification without forcing ACS work. Overrides do not grant trust or change unknown-data defaults. |

The saved comparison inputs and expected outcomes are unchanged. Tests compare all 29 reference Actions targets, including candidates that existing outer conditions prevent from executing. The four Prow groups below are accounted for through configuration and explicit gaps, rather than represented as unused Actions outputs.

### Current backup/restore variants

The current base splits database backup/restore into `e2e-db-backup-restore-tests-gke` and `e2e-db-backup-restore-tests-gke-konflux`. Each has its own matching label and image waiter. The ordinary variant carries the reference job's selection rules; the Konflux variant uses the same file rules and additionally requires `konflux-build` under enforcement. For example, `go.mod` plus `README.md` selects both, but only the ordinary variant can execute without the build label. An explicit Konflux job label alone also cannot supply images.

The pinned comparison tests map the reference's old job and label to the ordinary GKE variant. Twenty separate scenarios cover the current variants, including shadow mode, drafts, forks, unknown files, live label removal, and image tags. The old label is not a runtime alias. These additions follow the current base, rather than a newer comparison revision.

### Intentional differences from the comparison

The tests record 21 additions and 2 removals separately from the immutable comparison fixtures; the other 2,152 candidate results and all 83 E2E dispatch/waiter results retain the pinned behavior.

The unit workflow's overrides cover `Makefile`, `make/`, and `.github/actions/job-preamble/`. For example, `Makefile` plus `README.md` selects its unit-test candidates under enforcement. The pinned comparison selects only style for `Makefile`. An override belongs to its calling workflow: the unit workflow's match does not force E2E candidates.

Go changes outside the recognized CI-only trees select generated-file verification, because interface and generator changes can require regenerated mocks or strings. For example, `sensor/kubernetes/client/client.go` plus `README.md` selects that check. The comparison's broad/proto-only rule misses it. Existing outer conditions still apply, including the UI-only guard.

The two removals are ACS Go tests and Go lint for the saved `legacy selector files` scenario. Files under `ci/` and `tools/roxvet/` select their CI tooling tests instead of ACS Go checks. The exclusions also keep interface/CRD fixtures there from requesting ACS work. They apply per filename: adding `ci/native-selectors/file-facts/main.go` cannot remove the Go or integration votes from `pkg/concurrency/context.go`. Other Go tooling retains conservative ACS checks rather than treating every generator as CI-only.

Detector and unit-workflow edits do not produce a native run-all request. Their `selector` fact selects native regression tests. A CI-only diff covering those workflows, native-selector Go code, roxvet, and documentation selects style, Actions lint, selector tests, and roxvet tests. It does not select ACS unit, integration, benchmark, UI, or E2E work. No representative E2E job is requested. Explicit E2E labels and `ci-run-all-tests` still add coverage within their existing restrictions; shadow mode retains legacy execution.

An empty legacy `files` output is insufficient evidence of an override. It can also mean a detector edit, known empty diff, or failed collection. The separate `run_all` output records only an explicit caller-pattern match after revision and filename validation. Build-input matches are checked before the detector's legacy fallback, so adding a detector edit does not hide a real override. Classification can still fail afterward; the three QA/FIPS candidates then retain their skip default.

### Adding an unrelated file can remove a candidate

A PR changing only `central/sac/store.go` selects PostgreSQL. Add `sensor/kubernetes/client/client.go`: neither subsystem-only predicate holds, so neither PostgreSQL nor Sensor integration is selected by those predicates. Go and benchmark/lint checks still select. Add `pkg/concurrency/context.go`: the shared any-file rule selects both integration jobs again.

Likewise, `ui/package.json` selects UI candidates, but adding `README.md` removes the UI-only vote. `area/ui` restores the candidates. These are the fixed comparison's aggressive rules, not a recommendation that mixed PRs need less coverage. Keeping them visible tests whether engineers can understand and maintain them.

### A label cannot bypass an outer condition

On a docs-only PR, `area/ui` selects UI candidates, but the existing UI job condition can still skip the jobs because no UI path changed. Conversely, `ci-run-all-tests` on a UI-only PR does not bypass the non-UI outer guards on Sensor integration, CRD checks, and other guarded jobs. The prototype preserves these restrictions and tests candidates separately from execution.

Native Actions string comparisons are case-insensitive. The repository's existing lowercase label names are used by the fixtures; an invented differently cased label is not a claim of equivalence to the comparison's case-sensitive label lookup.

### Dependencies require repeated native conditions

`roxctl/main.go` selects the ordinary nongroovy E2E consumer. Its ordinary image waiter and image-build candidate must also select. Adding a new consumer label requires updating the consumer, the appropriate waiter, and its producer's candidate. Actions `needs` orders jobs but does not make that producer's condition true.

All builds remain enabled because Prow and manually requested tests also consume them. Skipping a build based only on the Actions dependency graph could leave `/test ocp-4-22-vm-scanning-e2e-tests` waiting for images that nobody builds. An inventory of all consumers is the smallest prerequisite for safely enforcing build skips; this prototype does not claim that audit is complete.

Build retains its existing event subscriptions. A new test label uses the images built for that PR head; it does not launch another image build. Its candidate summary describes the build event's label snapshot.

The build candidate is advisory and lives in `explain-image-build-selection`. That job waits for classification; the image matrix and producers do not. The report has no outputs that control the build jobs.

### Candidate, image permission and execution differ

On docs plus `e2e-qa-tests-gke-konflux`, enforcement selects the QA candidate but cannot execute it without `konflux-build`. Add that label and its Konflux waiter becomes eligible. Forks remain blocked in both modes. Under enforcement, a selected same-repository draft may execute, matching the comparison's deliberate change to legacy draft behavior.

Outside enforcement, an explicit Konflux job label can satisfy the legacy consumer rule while its waiter remains ineligible without the build label. The reusable child is then skipped. The prototype preserves this interaction instead of treating a selected flag as evidence of execution.

### Label changes and reruns have different snapshots

Adding or removing a label creates a new evaluation through `labeled`/`unlabeled`. For ordinary jobs, rerunning an older workflow uses the labels saved with that older event. For E2E, rerunning refreshes labels at the event head. For example, adding `e2e-byodb-tests` after opening can select BYODB on an E2E rerun; removing it removes that request.

A failed or stale-head label refresh disables E2E enforcement and falls back to legacy eligibility from event labels. It does not guess current labels. Uniform live-label semantics for ordinary jobs would require exposing a verified current label snapshot from an existing metadata read and changing their native expressions. That is a separate interface change, not something a path classifier should decide.

### Failed classification is different from a known empty diff

A known empty diff has complete facts with no positive matches. A failed API request, invalid path-group expression, failed classifier setup, stale head/base, or unrepresentable/oversized filename list has unavailable facts. With unavailable facts, PostgreSQL defaults to run, but the three Actions QA/FIPS jobs default to skip, even with `ci-run-all-tests`.

Classifier failure can leave legacy `ui_only` available; collection failure cannot. On a UI-only PR whose classifier fails, legacy non-UI guards can still skip jobs. On API failure, those guards fail open as before. Runner cancellation, invalid workflow YAML, or inability to start the detector can still block dependent jobs through Actions' normal dependency rules; a per-target fallback is not protection against a CI infrastructure outage.

### Skipped checks are not test results

Ordinary required jobs keep their IDs, matrix expansion, checkout and summary. Their expensive steps can skip and the check can succeed without executing tests. Selected E2E callers require successful eligible image waiters. An unselected PR suite instead invokes its reusable workflow with `should-run: false`, so the nested jobs can report skipped checks. It does not need images or start tests. Skipping the whole call would leave its nested required checks missing.

The current requirements snapshot covers 23 GHA contexts. Regression tests follow all four nested required E2E check paths through the reusable workflows and verify that they expand to skipped leaf jobs. They also cover image failures, missing or failed preparation, cancellation, and non-PR events across all nine callers. The pinned comparison's 22-context snapshot remains unchanged. Local expression tests cannot prove GitHub's check publication: the next CI run must confirm those four exact names. The separate required Prow UI check is outside this GHA fix.

## Prow: configuration, changed behavior and exact gaps

[The local patch](prow/native-selection.patch) changes four test entries in three pinned `openshift/release` source files. It does not edit generated jobs, deploy anything, or change execution commands, trust, branch, optional/required, timeout, postsubmit, or periodic configuration.

| Suite group from the comparison | Native prototype behavior |
| --- | --- |
| GKE QA | Replace the current `skip_if_only_changed: ^ui/` with a dependency/interface `run_if_changed`. Docs-only no longer schedules automatically; a broad change still schedules when mixed with unrelated files. |
| GKE nongroovy | Add the same broad selector to the main optional manual variant. This intentionally adds automatic scheduling on matching paths. Oldest/latest variants remain manual. The comparison gate itself did not start dormant jobs. |
| GKE UI | Keep the required main variant unconditional. Oldest/latest variants remain manual. Exact positive UI-only selection is unsupported by these native filters. |
| VM scanning | Add a feature/shared-dependency selector to OCP stable 4.18 and candidate 4.22. Other four existing variants remain manual. This intentionally adds automatic feature scheduling beyond the comparison's already-scheduled-job gate. |

The comparison's four suite IDs correspond to 15 existing aliases in the pinned configuration: three QA, three nongroovy, three UI, and six VM variants. Selecting all of them automatically would expand platform coverage. The prototype changes scheduling for the named variants above and leaves the others requestable through their existing explicit aliases.

### VM dependencies are broader than filenames containing “vm”

The path mapping covers the roxagent under `compliance/virtualmachines/`, Sensor discovery/VSOCK/scraping under `sensor/common/virtualmachine/`, Central services/storage/conversion under `central/`, and `pkg/virtualmachine/`. It also retains broad shared backend, Scanner, generated-interface, deployment/image, test-harness and CI setup paths, including roxctl and the roxie wrapper/version used during deployment. The suite enters through `tests/vm_scanning_suite_test.go` and `tests/vm_scanning_test.go`, uses `tests/vmhelpers/`, and is launched by `scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py` through `.openshift-ci/ci_tests.py` and `tests/e2e/run-vm-scanning.sh`.

A change to `pkg/concurrency/context.go`, `scanner/indexer/repositorytocpefetcher.go`, or a Central conversion file can affect the feature without a VM filename. These match. `sensor/common/virtualmachine/vmscraper/scraper.go` plus `ui/package.json` plus `README.md` also matches and automatically selects both covered variants. `unrelated/vm-name.txt` does not match.

The selector is conservative: `central/reports/README.md` also matches. It describes a broad dependency envelope, not a computed Go import graph or a minimal feature dependency list. Narrowing it needs evidence about transitive runtime, deployment, and harness dependencies. It is not counted as parity with the comparison, whose positive VM rules select only on broad interface/dependency changes or run-all.

### Path filters cannot switch individual PRs between shadow and enforcement

Docs-only QA demonstrates the conflict. Desired per-PR shadow behavior would report “QA would skip” and still execute QA; enforcement would skip it. The proposed native QA selector does not schedule that PR in either mode because it cannot read `ci-dispatcher-enforce`.

**The external patch is a configuration-level enforcement experiment, not shadow mode.** Exact per-PR switching would need broad scheduling plus an in-job label-aware gate, or a change to Prow's trigger mechanism. Neither is implemented. Accepting a repository-wide rollout would explicitly change the semantics.

### Positive every-file selection cannot be expressed by these selectors

The comparison selects UI for `ui/package.json` and stops selecting it when `OWNERS` is added. `run_if_changed: ^ui/` selects both. `skip_if_only_changed: ^ui/` excludes the first and permits the second. The native matcher runs when any file matches the run regex or any file fails the skip regex; adding a file cannot remove an existing reason to run.

Exact behavior needs an always-scheduled job that evaluates all files before executing, or changed selection semantics. The prototype implements neither workaround and preserves the required UI context by leaving it unconditional. Making UI conditional would also require coordinating GitHub's independently required check.

The same limitation leaves a nongroovy coverage gap. `roxctl/main.go` alone selects Actions `e2e-nongroovy-tests-gke`, but the proposed Prow `gke-nongroovy-e2e-tests` filter does not match it. Adding `README.md` removes the Actions roxctl-only match; the Prow filter matches neither diff. Adding `^roxctl/` to Prow would select both and change the mixed-diff behavior. The patch leaves this gap visible; `/test gke-nongroovy-e2e-tests` remains available for either diff.

### Arbitrary CI labels cannot schedule native Prow jobs

Docs plus `ci-run-all-tests` or `area/ui` does not start a filtered-out Prow job. `ci-dispatcher-enforce` also has no effect on the proposed selectors. The comparison's runtime gate could read labels only after Prow had scheduled its job; that was not automatic label-based scheduling either.

The existing extra-coverage control remains an explicit trusted request, such as `/test gke-qa-e2e-tests` or `/test ocp-4-22-vm-scanning-e2e-tests`. Automatic label requests would need a trusted label-event handler that issues equivalent native requests, checks the current head and avoids duplicate work. No such bot, service, bridge, or resolver is introduced here.

### Explicit requests, reruns and diff failures

Prow's explicit named request can bypass a path filter while retaining native authorization and branch checks. It also bypasses path collection, so it can recover from a changed-file API failure. `/test all` remains conditional and does not mean every manual variant; for example it does not select the narrowed QA job on docs-only paths. `/retest` considers failed contexts and eligible missing work through Prow's existing status/trigger logic.

This uses Prow's command routing directly. The comparison's gate reconstructs manual intent from recent comments and internal suite IDs, which can differ from external command aliases such as `ocp-4-22-vm-scanning-e2e-tests`. No comment-history evaluator is carried into this prototype.

The pinned upstream matcher experiment executes the force/path/error behavior. Trigger command routing, status-based reruns, authorization and webhook delivery still require real Prow access. Unlike the comparison's gate, the native matcher propagates automatic diff-fetch errors rather than applying per-suite run/skip defaults. Reproducing those defaults at scheduling time would require a controller fallback. Manual named requests remain the existing recovery mechanism.

## What the experiment establishes

Actions can express the fixed candidate rules and E2E dispatch without a central runtime plan, but needs shared file classification and repeated dependency predicates. Prow can automatically schedule feature tests, including mixed diffs, but cannot reproduce the same rollout, every-file and arbitrary-label behavior through its native path filters. The prototype keeps those differences observable. Passing local cases establishes neither production coverage sufficiency nor whole-system equivalence.
