# PR wall-clock wait

This is a temporary experiment for [PR #23602](https://github.com/stackrox/stackrox/pull/23602). The shared workflow publishes only for that PR in `stackrox/stackrox`. Remove this reporting integration before merging the prototype unless a separate decision retains it.

The bot comment reports one measurement and an expandable selection summary for each commit's initial GHA checks:

**Wall-clock wait = latest check completion − earliest workflow creation.**

This includes runner queues, dependency waits, failures, and green skips. Parallel durations are not added together. Timing reporters, canceled workflows/jobs, later label events, and manual reruns are excluded. The start is GitHub's workflow creation time, not the time of the push itself.

For example, workflows running from 12:00–12:10 and 12:05–12:15 produce a **15-minute** wait. A green-skip check finishing at 12:17 extends it to 17 minutes, just as a failed check would.

## How it works

Every PR workflow ends with `report-ci-timing`. Native `needs` lists every other job, and `always()` allows the call after failures or skips. The shared [reporting workflow](../../../.github/workflows/report-ci-timing.yaml) publishes only for `opened` or `synchronize` events on attempt 1 of a same-repository PR. Those conditions live in YAML, so label changes and reruns cannot extend a commit's measurement.

```mermaid
flowchart LR
  W[Initial jobs in each PR workflow] -->|needs + always| R[Shared bot reporter]
  G[Caller workflow creation and check completion timestamps] --> R
  R -->|Merge earliest start and latest finish| C[One comment with a row per commit]
```

Each reporter reads only its caller's first workflow attempt and job timestamps through `gh api`. It contributes that workflow's start and last non-canceled check completion. Failed checks, successful no-ops, and conditionally skipped checks count regardless of whether they allocated a runner. An unfinished check or missing timestamp causes an error instead of a misleading duration.

The comment stores `sha`, `start`, `end`, and workflow decision contributions in hidden JSON. Contributions merge by earliest start and latest finish. A workflow's repeated contribution replaces its decision summary rather than duplicating jobs. There is no scan of other workflows or historical reruns and no round-detection heuristic.

History retains at most **five measured commits**, ordered by their earliest reported workflow creation time. Older rows and their hidden records are removed together, including rows from force-pushed commits outside that window. Late contributions older than the retained time window are discarded.

Five commits alone do not guarantee that detailed job lists fit the **60,000-byte** budget. When necessary, the reporter omits decision details from the oldest retained rows first, keeping all retained timing rows and an explicit omission notice. Hidden decision records are removed too. Once omitted, a commit's details remain omitted so later workflow contributions cannot present a partial summary as complete; its timestamps continue updating. Even a single oversized decision report can therefore keep contributing timing updates.

**Values remain provisional while the initial checks and their reporters run.** Each completed workflow updates the same row. Once all initial workflows have reported, it represents the full GHA wait for that commit. If cancellation or a reporting error prevents a contribution, the comment can remain incomplete; it does not independently certify that every workflow reported.

All 23 current PR workflows have the final call; only workflows triggered by the initial opening/new-commit event contribute. A structural test checks every dependency list. Add a new job to its workflow's reporter `needs`; add the final call when adding a PR workflow. Existing test selection, image prerequisites, and required test checks retain their rules.

## Recorded selection decisions

Each timing row states how many recorded decisions permitted work, then lists exceptions with job links in three categories:

- **Skipped by selection:** the job's explanation recorded that enforcement rejected its candidate while execution restrictions permitted work.
- **Blocked by execution:** the explanation recorded an execution restriction, such as missing `konflux-build`. This category takes precedence when the candidate also did not match.
- **Explanation step present, decision unavailable:** an instrumented job's explanation did not run or succeed, or its decision log was unavailable, missing, malformed, or ambiguous.

The existing `Explain selection` action emits exactly one timestamped log record through GitHub's log capture: `STACKROX_CI_SELECTION_V1 permitted`, `skipped-by-selection`, or `blocked-by-execution`. Its native expression classifies the same inputs that control the action's execution output. The reporter reads that result; it never evaluates filename groups, labels, or job-selection expressions. Human-readable explanations remain available beside each job.

For E2E, nine adjacent preparation jobs run before the reusable test calls can skip. Each uses the same action with an optional `target` input containing its call's job ID. The action also emits `STACKROX_CI_SELECTION_TARGET_V1 JOB_ID`; its existing `run` output controls the call. The reporter uses the target to associate a recorded rejection with the skipped call and its nested jobs, rather than counting them again as unknown. Links lead to the preparation's explanation. The target is reporting metadata, not another selection rule.

Preparations run after their image waiters with `always() && !cancelled()`. They record failed prerequisites and execution restrictions before the test call is gated. Under enforcement, a successful image waiter that intentionally selected no work does not obscure a consumer's own selection rejection. For example, a docs-only PR with enforcement and `konflux-build` records E2E selection skips; a broad change without `konflux-build` records Konflux execution blocks. A preparation failure blocks its call and leaves its explanation unknown. Cancellations can prevent records and remain excluded from timing.

The reporter requests the first 1 MiB of logs only for jobs with a successful `Explain selection` step. A missing record beyond the excerpt is unknown, not a guessed decision. Log failures do not discard timing measurements. Skipped jobs without explanation steps, such as success-path Slack notifications and scheduled scanner maintenance, contribute only to a separate unattributed-skip count. Their statuses cannot establish selection reasons. Completed uninstrumented jobs are omitted; canceled jobs and timing reporters remain excluded.

Counts and lists cover only contributions received so far. Permission to execute does not prove that tests ran, and these lists do not estimate time saved. Label changes and reruns do not amend the initial-execution summary. Historical timing rows without decision contributions say that no decision summary was recorded.

The E2E preparations add nine runner starts and a possible failure before each test call. That execution overhead counts toward wall-clock wait. The temporary timing publisher's own overhead remains excluded.

For the first push containing this comment format, let older reporting workflows finish first. A reporter running older code can rebuild the same comment without retaining the decision fields it does not understand.

## One bot comment

The invisible marker `<!-- stackrox-native-ci-timing:v1 -->` identifies the comment. Only a matching comment owned by **github-actions[bot]** can be updated. Another hidden marker holds the timestamp data. Malformed data is rejected, and unchanged contents cause no edit.

GitHub job concurrency serializes comment creation and updates per PR. `queue: max` retains pending publishers. The automatic `${{ github.token }}` token supplies Actions/contents read and PR write permissions only to reporting jobs. Forks skip publication; drafts in the same repository can report. Failures in reporting are advisory. No personal token is used.

Reporting adds runner usage and delays workflow completion while publishing; log reads add requests for instrumented jobs. That overhead is excluded from the displayed wait. No polling, service, or default-branch workflow is needed. Actual bot permissions, concurrency, cancellation, and log retrieval still require a real PR run for each integration change.

## Interface and local validation

The Go helper accepts `--repo owner/repository` (default `stackrox/stackrox`), `--pr NUMBER`, and `--run WORKFLOW_RUN_ID`. Local execution previews the selected first attempt without publishing:

```sh
go run ./ci/native-selectors/timing-report --pr 23602 --run 39939959551
```

`--post` is supported only inside GitHub Actions with its workflow token. The shared workflow supplies the caller's `$GITHUB_RUN_ID`. The helper uses Go 1.26.7's standard library and the existing GitHub CLI; it adds no module and never selects or dispatches tests.

```sh
go test ./ci/native-selectors/timing-report
go vet ./ci/native-selectors/timing-report
go -C ci/native-selectors/testing test -run TestTimingWorkflowIntegration -v
```

Tests use simulated API responses and never post comments. They cover wall-clock aggregation, failures, green skips, cancellations, reporter exclusion, pagination, history merging, bot ownership, publication guards, and the native conditions excluding labels and reruns. Retention tests cover a seven-commit history, late contributions, oversized five-commit data, and continued simulated bot updates after a single oversized report. Decision tests cover all three exception categories, raw-log requests, invalid or duplicate records, safe job links, repeated contributions, and historical timing data. The workflow tests execute the actual explanation script for all eight combinations of candidate, eligibility, and enforcement, checking both its record and unchanged execution output.
