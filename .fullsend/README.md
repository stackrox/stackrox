# Fullsend PR review and fixes

[Fullsend](https://fullsend.sh/) runs on demand after this configuration is merged.
Comment one of these exact commands on an existing pull request:

| Command | Required repository access | Behavior |
| --- | --- | --- |
| `/fs-review` | `triage`, `write`, `maintain`, or `admin` | Run the standard reviewer |
| `/fs-fix` | `write`, `maintain`, or `admin` | Address feedback and push commits to the PR |

Fullsend checks the commenter's effective repository permissions through GitHub's
collaborator-permission API. Public read access and PR authorship are insufficient.
There is no separate team-membership check or additional authorization credential.

## Initial configuration

- Only human PR comments containing the exact commands above dispatch work.
  Automatic event triggers, schedules, and other commands are excluded.
- The `review` and `coder` roles support review and fix. The `code` agent remains
  explicitly disabled: new issue-driven coding, triage, prioritize, and retro
  workflows are outside the initial rollout. Code and fix share the coder App.
- Standard Fullsend behavior is retained. Reviews may approve, request changes,
  modify labels, or close rejected PRs. Fixes may change code on the existing PR.
- Routine status notifications are disabled. Extra issue-creation targets are
  empty; Fullsend still implicitly permits the source repository. Jira remains
  the work tracker.
- The workflow is pinned to v0.47.1 with read-only Actions cache access.

Enable additional agents or triggers through reviewed changes. Preserve the
built-in permission checks and preview upgrades with `--dry-run`. For an emergency
stop, set `kill_switch: true` in `config.yaml` and cancel active runs separately.

See the [configuration reference](https://github.com/fullsend-ai/fullsend/blob/c68903c0373e2216af03f5412642d4aa4019057f/docs/reference/config-reference.md),
[review guide](https://fullsend.sh/docs/agents/review), and
[fix guide](https://fullsend.sh/docs/agents/fix).
