# Testing native CI selection

Run from the repository root:

```sh
go -C ci/native-selectors/testing test -count=1 -v ./...
go -C ci/native-selectors/testing vet ./...
```

Use Go 1.26.7, Git, jq, and Bash 4 or newer. On macOS, set `NATIVE_TEST_BASH=/opt/homebrew/bin/bash` when the default Bash lacks `mapfile`.

The tests read conditions from the actual workflows. [`act` v0.2.89](https://github.com/nektos/act/releases/tag/v0.2.89) evaluates those expressions in supplied PR contexts. Its MIT-licensed dependency and supporting libraries stay in this test module. The full `act` workflow runner, Docker, Node, and npm are not used by these tests.

| File | Checks |
| --- | --- |
| `collector_test.go` | Real collector with simulated GitHub responses; classifier groups and invalid inputs. Large responses use a temporary file, avoiding process environment limits. |
| `actions_test.go` | All 83 saved scenarios, 2,175 candidate comparisons, and label, enforcement, draft/fork, failure, and image interactions. |
| `preservation_test.go` | Existing commands, matrices, outer conditions, triggers, required-check structure, and image prerequisites. |
| `compatibility_test.go` | The empty-label discrepancy below and 17 status/prerequisite cases. |
| `prow_test.go` | The unapplied external patch and native selectors; optional execution of five pinned upstream matcher methods. |
| `workflow_test.go` | Shared file loading, fixture contexts, process helpers, and expression evaluation. It contains no path or job-selection policy. |

The three comparison fixtures are pinned to #23403 at `34873a53e3b6`, with their bytes verified by `comparison.json`. Structural snapshots in `base-workflows.json` describe the original repository base `0a48209ff09e`; their hashes use sorted-key JSON from Go's `encoding/json`. Candidates and permission to execute are separate assertions. No test result proves that a product test ran.

## Empty-label compatibility boundary

GitHub's [`@actions/expressions` 0.3.0](https://github.com/actions/languageservices/tree/main/expressions) and `act` v0.2.89 disagree about an empty object filter. Given a PR with `labels: []`, or an event with no PR labels:

| Expression | GitHub expression library | `act` |
| --- | --- | --- |
| `toJSON(github.event.pull_request.labels.*.name)` | String `[]` | String `null` |
| `toJSON(github.event.pull_request.labels.*.name) == '[]'` | `true` | `false` |

For example, this native Actions condition recognizes an empty label list according to GitHub's library, while an `act` evaluation rejects it:

```yaml
if: ${{ toJSON(github.event.pull_request.labels.*.name) == '[]' }}
```

The current E2E metadata expression ends with `toJSON(github.event.pull_request.labels.*.name) || '[]'`. That fallback does not repair the discrepancy: the string `null` is nonempty and therefore truthy. The difference is recorded without normalization in `TestEmptyLabelCompatibility`. These expected GitHub values were checked against its published expression library, not through an additional live GitHub run.

Current selection conditions test label membership. Decoding either `[]` or `null` selects no label, so the saved candidate and dispatch decisions match. A future condition that compares against `[]`, or expects an array after decoding, can behave differently under `act`. Review this test when upgrading the evaluator; do not change scenario expectations to conceal a mismatch.

## Limits

`act` evaluates expressions; the Go tests still provide scenarios and assertions. Successful conditions do not establish live webhook delivery, fork approvals, branch protection, exact reusable-workflow check names, image availability, or Prow authorization and rerun routing. Those require real CI. The full `act` runner also [documents unsupported concurrency, permissions, cancellation, and summary behavior](https://nektosact.com/not_supported.html).

To execute the optional Prow matcher experiment, provide the source pinned in `../prow/source.json`:

```sh
NATIVE_PROW_JOBS_GO=/path/to/pinned/prow/pkg/config/jobs.go go -C ci/native-selectors/testing test -count=1 -v ./...
```

The experiment uses the five upstream methods unchanged and fixes only branch eligibility. It does not run the Prow controller or any cluster tests.
