# Fullsend review trial

This change prepares an on-demand review trial using Fullsend v0.47.1
(`c68903c0373e2216af03f5412642d4aa4019057f`). Jira remains the work tracker.
The trial retains Fullsend's standard reviewer and publishing behavior.
`kill_switch: true` currently keeps agent execution disabled while setup is
reviewed and validated.

## How Fullsend already checks repository permissions

Posting a command on a public repository does not by itself authorize an agent
run. For slash commands, the pinned reusable dispatcher:

1. Rejects bot invocations and identifies the comment's author.
2. Calls `is_authorized`, which delegates to `has_repo_permission` for that
   author. With this repository's configuration, it asks GitHub:
   `GET /repos/stackrox/stackrox/collaborators/{username}/permission`.
3. Checks the person's effective permission on this repository. GitHub includes
   directly granted access and inherited organization or team access. Fullsend
   also resolves custom repository roles from GitHub's effective permission flags.
4. Routes to the agent only if the command's required permission is satisfied.
   An unsuccessful permission lookup does not authorize dispatch.

| Upstream slash command | Minimum repository permission |
| --- | --- |
| `/fs-review`, `/fs-triage` | `triage`; `write`, `maintain`, and `admin` also qualify |
| `/fs-code`, `/fs-fix`, `/fs-retro`, `/fs-prioritize` | `write`; `maintain` and `admin` also qualify |

The table describes upstream authorization, not which commands this trial
exposes. Only `/fs-review` is enabled here. An ordinary public reader can view,
comment, and submit a PR without holding any of the required elevated roles.
PR authorship and a past merged contribution do not satisfy these dispatch checks.

Fullsend optionally accepts root `OWNERS` approvers/reviewers as an additional
permission source. This repository does **not** enable that provider. The
permission lookup is more precise than a comment's `author_association` value,
which is not used as proof of repository access.

This check controls agent dispatch after a workflow starts. GitHub may still
create a workflow run record for an arbitrary comment; that is distinct from
running an agent or using inference. Human invocation permission is also distinct
from the GitHub App permissions the agent receives to perform its role.

## Additional restrictions in this trial

These restrictions narrow the standard installation. They do not replace the
repository-permission check above.

| Restriction | Implementation | Reason for starting here | How it could be relaxed |
| --- | --- | --- | --- |
| Explicit review requests only | Custom workflow filter: exact `/fs-review` PR comment from a human; automatic event and prioritize entry points are absent | Keep execution intentional and avoid unsolicited work | Review a workflow change adding specific commands or events; keep the corresponding upstream authorization |
| Active `stackrox/eng` membership | Custom `authorize` job before the reusable dispatcher | Implements the additional team restriction requested in review | If repository permissions are accepted as the sufficient boundary, remove the team gate and its credential wiring while retaining Fullsend's own check |
| Review role only | Supported `roles` and `agents[].enabled` configuration; fix-stop handler absent | Evaluate PR review first; defer coding, fixes, and Jira-irrelevant issue workflows | Enable an individual role and its command in a reviewed change; install its App only for opted-in repositories |
| No additional issue-creation targets | Supported empty `create_issues.allow_targets` | Avoid automatic upstream GitHub issue filing | Add specific destinations when the workflow needs them |
| No routine status notifications | Supported `status_notifications` configuration | Limit start/completion notification noise | Enable selected comments or reactions |
| Execution initially disabled | Supported `kill_switch: true` | Separate merging the configuration from activating the trial | Set to `false` in a reviewed activation change after setup and live validation |
| Read-only Actions cache access | Workflow `cache-mode: read` | Prevent cache writes from comment-triggered runs | Change only after reviewing the need for write access and its trust implications |

The issue allowlist is not a blanket prohibition on GitHub issue creation:
Fullsend implicitly allows the source repository. Deferring triage, retro, and
prioritize avoids enabling their issue-oriented workflows; the standard review
role still has its normal GitHub permissions.

The team gate is stricter than Fullsend's repository-role boundary. Both checks
must pass: an eng member without the required repository permission is denied,
and a repository collaborator outside eng is denied by the additional gate.
GitHub's team-membership API includes child-team members.

## Standard reviewer behavior retained

This trial does not add a custom harness or publisher. The standard reviewer can
post reviews, approve, request changes, modify labels, and close a rejected PR.
It is not restricted to advisory comments. Existing human review requirements,
branch protections, and merge policy remain relevant when evaluating its output.

The generic reusable dispatcher retains its upstream job-token permission
contract. Selecting only the review role prevents other stages from being
selected; it does not turn every workflow credential into a read-only token.
A separately restricted publisher would be a future product/policy decision,
not a prerequisite imposed by this bootstrap.

## Setup, activation, and operations

The additional team gate requires an approved GitHub App installation in
`stackrox` with organization `Members: read`. Configure repository variable
`FULLSEND_AUTH_APP_ID` and secret `FULLSEND_AUTH_APP_PRIVATE_KEY`. Prefer a
suitable narrowly scoped App; its private key can mint the permissions the App
itself holds, even though this workflow requests only membership-read access.
The authorization token is short-lived, revoked at job completion, and never
forwarded to Fullsend. The authorization job checks out no repository code and
receives no inference credentials. Missing credentials or an unconfirmed
membership cannot authorize the dispatcher.

The normal repository `GITHUB_TOKEN` does not supply the organization membership
permission. This PR does not provision the authorization App or its credentials.
Vertex inference continues to use existing repository secrets
`FULLSEND_GCP_PROJECT_ID` and `FULLSEND_GCP_WIF_PROVIDER`; identifiers are not
duplicated in the committed config. An unused OpenAI API key is not forwarded.

Before activation, confirm the authorization App and review App installations,
inference access, trial owner, participating repositories, and run limits.
Demonstrate member/nonmember, failed lookup, disabled-command, and kill-switch
behavior in Actions, then evaluate the standard reviewer on agreed trial PRs.
Local mocked checks do not establish that these live integrations work.
Installing fewer Apps does not disable agents: keep configured roles and actual
App repository access aligned. Code and fix share the coder App. Coordinate any
App-access changes with other repositories using that installation.

Record useful findings, duplicate or incorrect feedback, author sentiment, and
cost. Public-repository Actions artifacts are not an internal-only reporting
channel; select storage and retention appropriate to the output being retained.

Use a pinned CLI checkout and `go run ./cmd/fullsend ... --dry-run` when preparing
upgrades. Review the generated changes: the upstream scaffold restores broader
event routes and the fix-stop handler. Reapply the deliberate StackRox workflow
restrictions before delivery. Supported configuration fields alone do not encode
the custom event filter or eng gate. Newer manifest-based management requires
reviewed adoption of this existing configuration; do not automatically converge
away these policy differences.

Relax restrictions through individual reviewed changes, rather than as a side
effect of a version upgrade. To stop a trial, set `kill_switch: true`, cancel
active runs, and restrict App repository access if needed. The kill switch stops
new dispatch; it does not cancel runs already executing.

## Implementation references

- [Pinned repository-permission helper](https://github.com/fullsend-ai/fullsend/blob/c68903c0373e2216af03f5412642d4aa4019057f/.github/workflows/reusable-dispatch.yml#L203)
- [Pinned slash-command routing](https://github.com/fullsend-ai/fullsend/blob/c68903c0373e2216af03f5412642d4aa4019057f/.github/workflows/reusable-dispatch.yml#L296)
- [Configuration reference](https://github.com/fullsend-ai/fullsend/blob/c68903c0373e2216af03f5412642d4aa4019057f/docs/reference/config-reference.md)
- [Mladen's team restriction](https://github.com/stackrox/stackrox/pull/23216#discussion_r4217789050)
- [Mladen's Jira feedback](https://github.com/stackrox/stackrox/pull/23216#discussion_r4217598200)
- [GitHub team membership API](https://docs.github.com/en/rest/teams/members#get-team-membership-for-a-user)
