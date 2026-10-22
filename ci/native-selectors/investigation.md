# Native CI selectors: investigation and discussion prototype

Investigated on 2026-10-22. This is a prototype for discussion, not a change proposed for production rollout.

## Executive summary

Native GitHub Actions conditions are a credible replacement for the current prototype's Go-based Actions selector and dispatcher. A small helper must collect changed-file facts because the pull-request event does not contain the complete diff. The job policy can then remain in ordinary Actions expressions. This implementation reproduces the pinned prototype's decisions across 83 local scenarios and preserves the existing workflow restrictions.

Prow's native selectors are useful, but they are not a complete replacement under the current StackRox configuration. They can prevent jobs from being scheduled, which saves work earlier than a gate inside a job. They also handle explicit `/test` requests without reconstructing the trigger from comment history. However, their file predicates cannot express the prototype's positive “every changed file matches” rules. Arbitrary CI labels do not trigger selection, and the required Prow UI check cannot simply disappear when irrelevant.

**Recommendation:** use native Actions conditions with a bounded diff helper, and consider native Prow filters for coarse scheduling. Keep a small Prow gate where exact per-PR preview and required-check behavior matter. Add a trusted label-to-command bridge only if labels must request Prow jobs automatically. This hybrid reduces custom machinery without claiming that static configuration covers the whole policy. If one policy, one explanation log, and future subgroup composition are priorities, retaining the resolver is reasonable. Its extra runtime code buys those capabilities.

The working prototype contains Actions workflow changes, executable local comparisons, and two reproducible patches for the external Prow source configuration. No external configuration has been deployed. The Prow label helper prints commands; it does not post them. Image builds remain enabled by default until all Prow image consumers have coordinated selection. These are explicit limits, not successful parity claims.

## Sources and pinned baseline

Read the [Granular Test Proposal](https://docs.google.com/document/d/1rpe0LZu39BLq9X_jUYH3iEWVmjqxkzIstQ8nwvMRv1I/edit), the design's [first tab](https://docs.google.com/document/d/1qYJFtJr7_19X9nPliYgClDs-nNRRYA8FPe1bAg04jN0/edit?tab=t.0), its experiments tab, and the [draft subgroup tab](https://docs.google.com/document/d/1qYJFtJr7_19X9nPliYgClDs-nNRRYA8FPe1bAg04jN0/edit?tab=t.zaickfmf0u7s). Neither document was edited. The implementation, rather than the broader proposals, defines parity here.

| Source | Revision used |
| --- | --- |
| [PR #23403](https://github.com/stackrox/stackrox/pull/23403), current head when inspected | `34873a53e3b60b5fc7b50b1446b02d11a1eb0fca` |
| Local prototype base, StackRox `master` | `0a48209ff09e12c09dd250cd7c8cab64bc851965` |
| External [openshift/release configuration](https://github.com/openshift/release/tree/3960ba6dc20f554e15fcf839e562a6d28353a2f3/ci-operator/config/stackrox/stackrox), default branch `main` | `3960ba6dc20f554e15fcf839e562a6d28353a2f3` |
| Upstream [Prow selector source](https://github.com/kubernetes-sigs/prow/blob/3f5cc7178d5a85a64fcf32c9cab401c937ee7218/pkg/config/jobs.go) | `3f5cc7178d5a85a64fcf32c9cab401c937ee7218` |
| OpenShift [ci-tools API](https://github.com/openshift/ci-tools/blob/4a4f06677f7028a47c96cbb032aad57f39100125/pkg/api/types.go) | `4a4f06677f7028a47c96cbb032aad57f39100125` |

The Prow revision is the source used for the matcher experiment. It is not a verified version of the deployed OpenShift controller. Official documentation was checked on the investigation date. The source hashes, job inventory, external job aliases, and required-check snapshot are retained beside this document.

### What the pinned prototype implements

The [policy](https://github.com/stackrox/stackrox/blob/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca/ci/test-domains.yaml) has **33 logical targets**, including image and helper targets. That is not the inventory of every StackRox CI job. A positive run vote wins over a skip vote. The final `remaining` rule skips all undecided targets. Consequently, for this particular configuration, selection equals the union of positive rules followed by dependency expansion. The numerous negative rules do not remove any positive selection. Native expressions can encode that union without implementing a general rule interpreter.

The aggressive fallback for *unmatched paths* is a policy choice: `Makefile` or `OWNERS` alone selects only `style-check`. It is different from a *resolution failure*, which uses [decision-defaults](https://github.com/stackrox/stackrox/blob/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca/ci/decision-defaults). Those defaults run all enrolled targets except the three QA/FIPS Actions targets, Prow GKE QA, and Prow VM scanning. The proposal's broader conservative selection goals must not be attributed to this implementation.

The [Actions adapter](https://github.com/stackrox/stackrox/blob/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca/ci/select/gha.go) uses a three-dot Git diff. Ordinary gates use event labels; E2E dispatch refreshes labels from GitHub. Without `ci-dispatcher-enforce`, gates report the plan but preserve existing execution. With enforcement, E2E dispatch uses selected targets, still excludes forks, and still requires `konflux-build` for Konflux PR jobs. Other workflows retain their existing outer conditions.

The [Prow adapter](https://github.com/stackrox/stackrox/blob/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca/ci/select/prow.go) gates jobs that have already been scheduled. Its four enrolled suite IDs are `gke-nongroovy-e2e-tests`, `gke-qa-e2e-tests`, `gke-ui-e2e-tests`, and `ocp-vm-scanning-e2e-tests`. External variants can pass these IDs through `TEST_SUITE`. Setting a label does not itself schedule dormant jobs. The [entrypoint gate](https://github.com/stackrox/stackrox/blob/34873a53e3b60b5fc7b50b1446b02d11a1eb0fca/.openshift-ci/dispatch.sh) runs before its credential import and GKE setup, but an OpenShift workflow can provision its cluster in an earlier step.

For `openshift/release` rehearsal PRs, the baseline also recognizes `/label: ci-dispatcher-enforce` in the PR body. Its diff uses the checked-out `HEAD`, while the comment override uses the original PR head's committer timestamp. The native patches have no corresponding per-rehearsal enforcement switch.

The subgroup tab proposes additional policy and runner contracts. It is not implemented at this SHA. Neither subgroup routing, generalized platform deduplication, nor architecture selection is included in this comparison's parity target.

## GitHub Actions: strongest practical design

### Workflow filters are useful but insufficient here

Native `paths` patterns select a workflow if a changed path matches; ordered `!` patterns exclude or re-include paths. Branch and path filters must both match. They do not provide an independent label override for a filtered-out PR workflow. A workflow excluded by a filter can leave a required check pending. The current documentation specifies a **3,000-file** diff limit, with workflow execution on diff timeouts or pushes over 1,000 commits. [Workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onpushpull_requestpull_request_targetpathspaths-ignore)

For example, `paths: ['ui/**']` could select the existing `ui` workload on UI changes, but cannot reproduce “UI only,” `area/ui`, and the current required-check contract together. Splitting off a second label-triggered workflow introduces duplicate check names, duplicate runs, and coordination around dependencies. It is possible, but less maintainable than one unfiltered workflow with job conditions.

### Collect facts once per workflow; express policy in YAML

The reusable `native-ci-selection.yaml` workflow checks out the tested PR head, collects all paths with `git diff --name-only -z BASE...HEAD`, and evaluates native expressions. The Python helper returns only named `any` and `every` predicates. It knows nothing about jobs, labels, precedence, dependencies, or enforcement. Empty diffs do not satisfy `every`. Invalid inputs fail the collection step instead of masquerading as an empty diff.

This is additional runtime code, but it is not a replacement resolver hidden in Python. Job decisions are visible in Actions expressions. Dependency requirements are expanded statically into the expressions for image/helper targets. A tiny serializer writes already evaluated booleans to outputs; it does not interpret policy.

For example, the positive rule for the existing `sensor-integration-tests` target is:

```yaml
SELECT_sensor_integration_tests: >-
  ${{ (steps.paths.outputs.known == 'true' && (
    steps.context.outputs.all == 'true' ||
    contains(fromJSON(steps.context.outputs.labels), 'area/sensor') ||
    contains(fromJSON(steps.paths.outputs.every || '[]'), 'sensor') ||
    contains(fromJSON(steps.paths.outputs.any || '[]'), 'shared') ||
    contains(fromJSON(steps.paths.outputs.any || '[]'), 'central-communication') ||
    contains(fromJSON(steps.paths.outputs.any || '[]'), 'sensor-communication')
  )) || (steps.paths.outputs.known != 'true' && true) }}
```

`all` means a `ci-run-all-tests` label or a broad path such as `go.mod`, `go.sum`, `proto/`, `generated/`, or a `.proto` file. The final clause implements this target's failure default. Overlapping positive predicates combine with OR, so a UI Go file still selects Go checks. Native `contains` is case-insensitive; the resolver's string matching is exact. The evaluated policy assumes the existing, canonically named labels, not arbitrarily recased replacements. [Expression reference](https://docs.github.com/en/actions/reference/evaluate-expressions-in-workflows-and-actions#contains)

The existing consumer retains its outer condition:

```yaml
sensor-integration-tests:
  needs: [detect-changes, native-selection]
  if: >-
    ${{ !cancelled() && needs.detect-changes.result == 'success' &&
        !needs.detect-changes.outputs.ui_only &&
        (needs.native-selection.outputs.enforced != 'true' ||
         needs.native-selection.outputs.sensor-integration-tests != 'false') }}
```

A missing selector output permits execution when the existing condition permits it. Explicit dependency-result checks prevent an unrelated failed selector job from suppressing all consumers through Actions' default success condition. Cancellation and failures of existing prerequisites still matter. [Job dependencies](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-jobs)

### Preserve checks and cross-system prerequisites

A skipped ordinary job reports success. However, a job condition is evaluated before matrix expansion. Gating an entire matrix risks losing the exact required variant names. This prototype therefore keeps the build and Go matrices and gates their steps. It also keeps E2E reusable calls and the image-wait jobs present, passing the existing `should-run` flag to children. This costs lightweight runner allocations even when test work is skipped. [Job conditions](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-jobs-with-conditions), [matrix evaluation](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idif)

The observed default-branch ruleset requires 22 Actions contexts and `ci/prow/gke-ui-e2e-tests`, with distinct integration IDs. A green check with an invented name or from the wrong integration does not replace them. The [ruleset snapshot](testdata/required-check-evidence.json) and [Actions mapping](testdata/required-gha-checks.json) retain this evidence. The ruleset is live configuration and may change independently of the repository.

The pinned `requires` graph connects five ordinary Actions E2E targets to `wait-for-images`, then to `should-dispatch` and `build-images`. Three Konflux targets require their separate waiter. It does **not** connect Prow image consumers to `build-images`. Prow jobs use images through the existing [CI image helpers](https://github.com/stackrox/stackrox/blob/0a48209ff09e12c09dd250cd7c8cab64bc851965/scripts/ci/lib.sh). A candidate plan that skips builds while Prow still runs is not a usable execution plan.

For this reason, actual image skipping additionally requires repository variable `CI_NATIVE_ALLOW_IMAGE_SKIPS=true`. Leave it unset for this prototype. The selection outputs still expose the baseline decision for comparison. Enabling this variable requires coordinated coverage of **all** Prow image consumers, not just the four enrolled suites. This is an intentional execution difference that reduces achievable savings.

### Labels, manual runs, and configuration changes

The modified PR workflows subscribe to `labeled` and `unlabeled`, as well as their existing lifecycle events. Adding or removing `ci-dispatcher-enforce` starts a new evaluation. Ordinary workflows use that event's labels. E2E dispatch fetches current labels, and disables enforcement if that fetch fails. Existing concurrency groups can cancel an older run when a label event starts a newer one. No helper cancels individual Prow jobs when labels are removed. [PR event activities](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#pull_request)

Users request additional Actions E2E work with existing job-name labels such as `e2e-byodb-tests`. Under enforcement, `ci-run-all-tests` selects all 33 targets but does not override outer UI filters, fork restrictions, or Konflux availability. In preview mode it does not turn all legacy dispatcher decisions on. A manual dispatch is a distinct event and uses existing non-PR behavior where supported. Re-running a PR workflow retains its original event, SHA, and ref; it is not an unconditional override. A rerun of the dispatcher can nevertheless see newer labels through its explicit API refresh. [Rerun behavior](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/re-run-workflows-and-jobs)

Existing merge pushes, nightly tags, workflow calls, and manual build inputs retain their triggers and execution rules. The prototype adds no new scheduling or merge-queue event. Fork token restrictions and approval requirements still apply; no privileged `pull_request_target` execution was introduced. [Fork execution](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflows-in-forked-repositories)

Policy changes take effect with the workflow/helper revision used by a run. Changes to this prototype's own paths trigger diagnostics, and its small regression workflow runs on every PR event it subscribes to. It is deliberately broader than the baseline `selector-tests` candidate. The summaries show facts and candidate booleans, but do not reproduce the resolver's rule-by-rule trace or its bot comment.

### Failure boundaries

Ordinary baseline gates allow work if the selector binary cannot be built. The baseline E2E dispatch bootstrap is different: a failed bootstrap step can block dispatch before its Go fallback logic runs. This prototype removes that build dependency and allows legacy E2E dispatch if the reusable selection job fails. That is an intentional fallback improvement, not exact failure parity.

A failed diff or malformed pattern produces unknown facts and uses the embedded target defaults. A failed live-label lookup disables E2E enforcement. Neither design can recover from malformed workflow YAML that prevents GitHub from creating the workflow. In Prow, native path-fetch errors propagate instead of applying the prototype's target defaults; an explicit `/test JOB` bypasses that path lookup. Automated fail-open scheduling would require another controller or gate.

## Prow: strongest practical design

### Select before scheduling

Prow supports `run_if_changed`, `skip_if_only_changed`, `always_run`, branch filters, explicit triggers, and rerun commands. In the evaluated configuration, the two path fields are mutually exclusive and cannot accompany `always_run: true`. Branch eligibility still applies. A normal `run_if_changed` job runs when any file matches; a `skip_if_only_changed` job runs when any file does not match. Alternation combines positive paths within a regex. [Selector implementation](https://github.com/kubernetes-sigs/prow/blob/3f5cc7178d5a85a64fcf32c9cab401c937ee7218/pkg/config/jobs.go), [configuration validation](https://github.com/openshift/ci-tools/blob/4a4f06677f7028a47c96cbb032aad57f39100125/pkg/api/types.go)

The external [StackRox source configuration](https://github.com/openshift/release/blob/3960ba6dc20f554e15fcf839e562a6d28353a2f3/ci-operator/config/stackrox/stackrox/stackrox-stackrox-master.yaml) currently makes GKE UI unconditional and required. GKE QA is optional and excluded only for UI-only changes. GKE nongroovy, oldest/latest variants, and the six OpenShift VM variants are manual. The generated [presubmits](https://github.com/openshift/release/blob/3960ba6dc20f554e15fcf839e562a6d28353a2f3/ci-operator/jobs/stackrox/stackrox/stackrox-stackrox-master-presubmits.yaml) supply actual command aliases, contexts, and master branch expressions.

For the current policy, GKE QA, Prow nongroovy, and VM scanning have positive file votes only for broad changes. This exact selector expresses that file portion:

```yaml
- always_run: false
  run_if_changed: ^go\.(mod|sum)$|^proto/|^generated/|\.proto$
  as: gke-qa-e2e-tests
  optional: true
  steps:
    workflow: stackrox-stackrox-e2e-job
```

Do not replace this with a plausible “all backend paths” regex and call it parity. The pinned map does not select Prow GKE QA for an ordinary backend Go change. Changing that coverage may be desirable, but it is a separate policy decision.

Two generated patches make the scheduling trade-off reviewable:

| Patch | Effect |
| --- | --- |
| [keep-triggers.patch](prow/keep-triggers.patch) | Changes only automatic GKE QA to the broad selector. Keeps dormant jobs dormant. This is the closer baseline comparison. |
| [schedule-selected.patch](prow/schedule-selected.patch) | Also adds the selector to manual GKE QA/nongroovy variants and six VM jobs: 12 source entries total. This makes broad changes schedule selected work, which is an explicit extension beyond the baseline gate. |

Both preserve job names, test steps, `optional`, timeouts, and non-PR entries. Both retain the required UI job as `always_run: true`. Manual oldest/latest UI variants remain manual because their positive “UI only” condition is not expressible by these selectors. A 15-alias inventory covers all external variants of the four enrolled suite IDs; unrelated suites remain outside this prototype.

These are patches to ci-operator **source** configuration. They must be followed by `make jobs`, the release repository's configuration checks, and rehearsal of affected jobs. Editing generated presubmits alone would drift from their source. No external repository edit or rehearsal was performed. [Release contribution workflow](https://docs.ci.openshift.org/how-tos/contributing-openshift-release/)

### Demonstrated boundary: positive “every file matches”

With no labels, the baseline selects Prow UI for a change to `ui/package.json`, but not for `ui/package.json` plus `OWNERS`. Either native path selector is monotonic under adding changed files: an existing matching file, or an existing nonmatching file, remains present. Therefore no single ordinary Prow `run_if_changed` or `skip_if_only_changed` predicate can reproduce that transition. OR-ing duplicate job entries does not fix it. This is a demonstrated constraint of those mechanisms under a fixed branch and ordinary automatic trigger, not a claim about everything Prow could run.

The smallest exact workaround is an always-scheduled UI job with a small runtime gate that checks all paths and labels, then reports success when skipping. That still incurs scheduling and possibly setup costs. Another option is to broaden UI selection to any UI path and change the required-check contract so irrelevant jobs can be absent. Neither option is implemented silently here: the native prototype retains unconditional UI execution.

### Labels need a trusted event bridge

The evaluated [trigger handler](https://github.com/kubernetes-sigs/prow/blob/3f5cc7178d5a85a64fcf32c9cab401c937ee7218/pkg/plugins/trigger/pull-request.go) handles special trust labels such as `ok-to-test` and `lgtm`; it does not offer a general `ci-run-all-tests` or `area/ui` selector. Removing an arbitrary label does not recompute scheduled work.

The bounded helper [label_requests.py](prow/label_requests.py) demonstrates the missing mapping. It reads saved PR labels and current-head statuses, then prints explicit commands. With the closer trigger profile, `ci-run-all-tests` requests the two normally automatic enrolled suites; `area/ui` requests UI. With `--schedule-selected`, run-all requests all 15 enrolled aliases, deliberately expanding scheduling. It deduplicates overlapping labels and avoids restarting checks recorded as pending or successful. Its CLI declines closed, draft, and fork PRs; trusted manual Prow commands remain available for those cases according to Prow's own rules.

To automate this plan, a bridge must read current PR state, validate the current SHA, select the latest status per context, deduplicate deliveries, and post as an identity trusted by Prow. StackRox's [trusted-app configuration](https://github.com/openshift/release/blob/3960ba6dc20f554e15fcf839e562a6d28353a2f3/core-services/prow/02_config/stackrox/stackrox/_pluginconfig.yaml) lists Dependabot, Konflux, and the OpenShift merge bot, not `github-actions`. A default Actions-token comment is therefore not a demonstrated solution. This operational integration is unfinished. It is smaller than a path/dependency resolver, but requires ownership, credentials, and failure handling.

The helper does not reproduce the baseline perfectly. For example, baseline run-all does not schedule jobs, and a successful no-op gate can hide that tests never executed. Status-based deduplication alone cannot distinguish that no-op success from a completed test. A deployed bridge would need execution metadata or a controlled rerun policy. Labels should request additional work, not cancel useful running jobs.

### Native commands are better than reconstructing comment history

Explicit `/test gke-qa-e2e-tests` or `/test ocp-4-22-vm-scanning-e2e-tests` bypasses the native path predicate while preserving branch and trust checks. `/test all` remains conditional and does not mean “every configured optional/manual test.” `/retest` handles failed work and eligible missing work. The newer `/test-manual-required` command concerns required manual jobs, not these optional VM jobs. [Trigger command implementation](https://github.com/kubernetes-sigs/prow/blob/3f5cc7178d5a85a64fcf32c9cab401c937ee7218/pkg/plugins/trigger/generic-comment.go)

By comparison, the pinned gate scans comments newer than the head commit's committer timestamp for `/test all` or its internal suite name. A historical comment can keep overriding later reruns. An alias such as `ocp-4-22-vm-scanning-e2e-tests` differs from internal ID `ocp-vm-scanning-e2e-tests`, so a genuine explicit request can be missed. Native triggering avoids that mismatch. Its `/test all` behavior is nevertheless different: it can exclude the broad-selector QA job on a docs change where the current external selector plus baseline comment override would run it. Use the explicit command when requesting extra coverage.

Prow's skip plugin can clear stale nonblocking statuses, and administrators have override mechanisms. These are manual recovery tools, not a replacement for selection or evidence that a required test ran. [Prow job and status semantics](https://docs.prow.k8s.io/docs/jobs/)

### Required checks, preview, and wider platform options

Prow/Tide can treat a conditional job as required only when applicable. That does not remove an independently configured GitHub requirement. StackRox marks branch protection [unmanaged](https://github.com/openshift/release/blob/3960ba6dc20f554e15fcf839e562a6d28353a2f3/core-services/prow/02_config/stackrox/stackrox/_prowconfig.yaml), and its current GitHub ruleset explicitly requires the Prow UI context. Making that job conditional without changing the contract could leave PRs waiting. Keeping UI unconditional is a current-configuration compromise, not a Prow platform limitation.

The external selectors apply repository-wide once deployed. They cannot themselves switch between preview and enforcement using `ci-dispatcher-enforce` on an individual PR. Keeping the current worker gate allows that behavior; alternatively, accept a configuration-level rollout. The supplied patches represent enforcement and must not be mistaken for shadow-mode configuration.

Two broader native options were also examined:

- **In-repository Prow configuration** can version job definitions with code. It needs central enablement and a configuration-validation job. The inspected central config has allowed clusters but no enablement for StackRox. It addresses distribution, not the missing label or positive-all predicate. [Inrepoconfig](https://docs.prow.k8s.io/docs/inrepoconfig/)
- **OpenShift Pipeline Controller** supports manual, automatic, and LGTM-triggered second stages, plus `pipeline_run_if_changed` and `pipeline_skip_if_only_changed`. It can supply staged dispatch and missing-job handling without a StackRox dispatcher. StackRox is absent from both pinned [main enrollment](https://github.com/openshift/release/blob/3960ba6dc20f554e15fcf839e562a6d28353a2f3/core-services/pipeline-controller/config.yaml) and [LGTM enrollment](https://github.com/openshift/release/blob/3960ba6dc20f554e15fcf839e562a6d28353a2f3/core-services/pipeline-controller/lgtm-config.yaml). Enrollment and dispatch-gate requirements would be a broader change. Its traditional filters do not supply the missing predicates. The documented closed-beta agentic mode introduces an external test selector, so it is not evidence that native filters solve this policy. [Pipeline configuration](https://docs.ci.openshift.org/how-tos/creating-a-pipeline/)

## Behavior comparison

Categories classify the evaluated mechanism's capability, not whether this local prototype has deployed it: **1** directly supported; **2** additional configuration or a small helper; **3** substantial complexity or changed semantics; **4** unsupported by the stated native mechanism. A category 4 row names the smallest alternative outside that mechanism.

| Behavior | Pinned prototype | Actions approach | Prow approach |
| --- | --- | --- | --- |
| Any matching path; overlapping positive votes | Run wins; positive union | **2:** full diff facts plus OR expressions | **1:** regex alternatives for enrolled broad selectors |
| Positive every-file predicates; mixed areas | All-file votes can disappear when unrelated paths are added | **2:** generic `every` facts; exact candidates tested | **4:** ordinary native path predicates cannot express this; retain a runtime gate or broaden semantics |
| Area labels and explicit job labels | Add run votes; old outer guards still apply | **1:** native label expressions | **2:** trusted label-to-command bridge; planner only implemented |
| `ci-run-all-tests` | Selects 33 targets, not all CI; does not bypass guards or schedule dormant Prow jobs | **2:** expressions and static dependency closure preserve candidates | **3:** explicit commands can request more jobs, but trigger population and status deduplication differ |
| Image/helper dependencies | Explicit small graph; omits Prow consumers | **2:** native `needs`, expanded image predicates, preserved contexts | **3:** no cross-system dependency graph; retain builds or coordinate producers and consumers |
| Draft/fork restrictions | Existing restrictions; enforced Actions E2E can run on same-repo drafts, never forks | **1:** existing expressions and permissions | **1:** native draft/trust/manual rules; no restrictions removed |
| Manual runs and reruns | Event-specific behavior; Prow infers manual intent from comments | **1:** preserve existing dispatch inputs; reruns are not force-all | **1:** explicit `/test` handles force and aliases; `/test all` has changed semantics described above |
| Labels changed after opening | Actions rerun subscriptions; Prow reads labels only once scheduled | **1:** labeled/unlabeled events; **2** for live label refresh | **2:** event bridge needed to schedule on arbitrary labels |
| Merge/nightly/non-PR execution | Outside enforced PR selection | **1:** existing triggers and rules retained | **1:** patches exclude postsubmit/periodic entries |
| Per-PR preview/enforcement | `ci-dispatcher-enforce` controls gates | **1:** conditions; summaries expose candidates | **4:** static native presubmit selectors lack this input; a worker gate or configuration rollout is needed |
| Diff/resolver failure | Defaults; unreadable ownership can permit work; dispatcher bootstrap can fail | **2:** safe missing-output conditions and embedded defaults; malformed workflow YAML still fails | **3:** matcher errors propagate; explicit commands bypass paths, but automatic fail-open needs another mechanism |
| Required checks for skipped work | Existing jobs can report successful no-ops | **2:** keep matrix variants and reusable child checks | **3:** leave UI unconditional, retain a gate, or change GitHub requirements |
| Explanation and preview comments | Decision log, summaries, bot comment with stale-head guard | **2:** summaries implemented; equivalent comment publisher omitted | **2:** command/config evidence; unified per-PR explanation needs a reporter |
| Configuration changes | Shared runtime map; external Prow triggers still separate | **2:** source-controlled expressions/facts and regression tests | **2:** generate source patches, regenerate Prow jobs, check drift; controller update required |
| Proposed subgroup selection | Not implemented | **3:** possible inputs/matrices, but needs runner contract and union rules | **3:** needs parameter handoff or more jobs; native presubmit regexes select jobs, not tests within them |

### Representative outcomes

These examples use existing StackRox paths. “Candidates” excludes `style-check`, which is always selected, and describes the enforced resolver decision before outer workflow restrictions. The checked scenarios include corresponding path classes and label combinations.

| Diff or labels | Baseline and native Actions candidates | Practical consequence |
| --- | --- | --- |
| `README.md` or `Makefile` | No other candidates | Does not implement the proposal's suggested broad Makefile fallback. Image work is retained by this prototype's safety setting. |
| `ui/package.json` | `ui`, `ui-component`, `gke-ui-e2e-tests` | Native Prow UI remains unconditional rather than reproducing the gate. |
| `ui/package.json` + `README.md` | No other candidates | Adding a docs change defeats the positive UI-only rule. This surprising baseline behavior is preserved in candidates. |
| `central/sac/root_scope_checker_core_context_enricher.go` | `go`, `go-bench`, `golangci-lint`, `go-postgres` | Central-only rule selects PostgreSQL. |
| Same Central file + `sensor/kubernetes/client/client.go` | `go`, `go-bench`, `golangci-lint` | Neither all-file subsystem predicate holds. The union is of matching rules, not of independent per-file domain plans. |
| `pkg/concurrency/context.go` | Above Go checks plus PostgreSQL and Sensor integration | Shared any-file rule selects both subsystems. |
| `roxctl/main.go` | Go checks, local roxctl, GHA nongroovy, image waiter/build/dispatch prerequisites | Static dependency expansion preserves this closure. |
| `go.mod` or any `.proto` path | All 33 | Native broad Prow filter matches. The closer profile still leaves manual jobs dormant. |
| Docs + `area/sensor` + `area/ui` | Sensor integration and three UI targets | The existing Actions UI path guard can still skip UI jobs. |
| Docs + `ci-run-all-tests` + `konflux-build` | All 33; Konflux eligible | Full candidate set still does not guarantee every outer condition passes. |

`ci-all-qa-tests` controls the tests inside an already-running QA workload. It does not schedule QA by itself. The current [QA runner](https://github.com/stackrox/stackrox/blob/0a48209ff09e12c09dd250cd7c8cab64bc851965/qa-tests-backend/scripts/run-part-1.sh) normally uses BAT tests on PRs and honors that separate label, with its existing rehearsal exception. Adding `ci-run-all-tests` alone does not mean full QA. The subgroup extension must preserve this distinction and define how group sets combine before either native or custom selection can implement it reliably.

## Validation and results

| Validation | Result and scope |
| --- | --- |
| Pinned Go implementation | Executed its own decision and dispatcher functions in a temporary source archive at the recorded SHA. Regenerated and compared all 83 saved outcomes. No Git worktree was created. |
| Actual Actions expressions | Executed with GitHub's `@actions/expressions` evaluator against those outcomes. Covers paths, mixed areas, every area/job label, overlapping labels, run-all/QA/Konflux, drafts, forks, preview, unknown diffs, push, and manual event contexts. All passed. |
| Workflow wiring | Evaluated 168 step gates; checked label subscriptions, failed/missing selector output, live-label refresh failure, image-readiness behavior, waiter/child topology, and 22 required Actions contexts. Passed. |
| Build matrices | Executed the existing matrix-construction shell with fixture PR/label/tag functions for ordinary PR, expanded PR, merge, and nightly tag. Required build variants remain present. No image build ran. |
| Path helper | Six Python test methods across both helpers passed. Includes empty/mixed/overlapping predicates, malformed patterns, missing SHAs, a 4,001-path input, and a real temporary Git diff with deletion, rename, and newline filenames. The large-input test evaluates the helper, not GitHub's server-side limit. |
| Prow matcher | Executed 12 cases using unmodified methods extracted from the pinned upstream source. Covers mixed paths, empty diffs, dormant manual jobs, explicit-force bypass, path-fetch failure, and always-run behavior. Branch matching is a fixed fixture; no webhook/controller was run. |
| External patch validation | Both patches apply to the pinned source snapshot. Parsed before/after structures show 1 and 12 selector-only changes, respectively, with execution and non-PR fields preserved. |
| Workflow lint | Repository-selected `actionlint.kjanat.dev` v1.17.0 passed for all eight changed/new workflows. Whitespace validation passed. |
| External integration | Trust handling, comment posting, Prow generated-config checks/rehearsal, real GitHub check creation, and cluster execution were not executed. No CI-duration or cost savings were measured. |

The 83 scenarios establish candidate and dispatcher parity for representative inputs, not a proof of whole-system equivalence. Cases named “manual” evaluate existing non-PR dispatch logic; they do not claim that every workflow exposes a manual trigger. Tests of context names and matrix variants cannot prove GitHub's eventual reporting behavior. The [README](README.md) contains reproducible commands and separates routine tests from experiments requiring the pinned external inputs.

Before any adoption, test a real same-repo draft, a ready PR, and an approved fork. Add/remove enforce, run-all, area, and Konflux labels at the same head; check both reruns and new commits. Verify exact required contexts, including successful no-op matrix and reusable jobs. Rehearse the external patches and exercise explicit `/test`, `/test all`, `/retest`, and a diff-fetch failure. Keep image skipping disabled until the producer/consumer audit is complete. These are CI-access validation tasks, not completed results.

## Maintenance trade-offs

The native Actions version removes a Go build from decision execution and makes conditions inspectable in workflow YAML. It still owns a diff collector, a pattern vocabulary, repeated output declarations, embedded failure defaults, and static dependency closure. Adding an E2E job requires updating its condition, image/helper predicates, wiring, and fixtures. Actions does not infer that selecting a consumer should select a producer.

Patterns are centralized for Actions, but broad paths are duplicated in Prow configuration. The two CI systems also have separate job inventories and aliases. The patch generator checks source hashes and refuses unexpected upstream drift; it does not resolve it. The saved oracle pins the comparison and will not automatically approve a future policy change. Reviewers must deliberately update fixtures and explain changed outcomes.

A build-time generator could turn one declarative policy into Actions expressions and Prow regexes, with checked-in output and a drift check. That would be a compiler, not a per-run resolver. It could eliminate mechanical duplication, but would need validation for unsupported predicates, fallback behavior, dependency expansion, and generated job names. For the current 33-target experiment, handwritten expressions plus tests are easier to audit than adding that compiler immediately.

The small helpers have distinct costs: the path collector runs once per calling workflow; label refresh adds one API read for E2E dispatch; the optional label bridge needs trusted delivery and idempotency; external patch generation depends on PyYAML only during development. The expression evaluator, YAML parser, and bundler are pinned MIT-licensed test dependencies, not runtime CI policy dependencies. PyYAML is also test/tooling-only. The Go oracle uses the baseline's Go 1.26.7 module.

Native configuration can replace much of the current implementation, but a promise of exact cross-system behavior would still require custom integration. The useful decision is which behavior deserves that cost. Retaining a small Prow gate for required checks and rollout control is defensible; implementing a general shared resolver becomes more attractive if subgroup composition and detailed explanations become firm requirements.
