---
name: babysit-ci
description: Babysit CI on a StackRox PR: reactively watch both GitHub Actions and Prow checks, react to each result as it lands, triage failures, and summarize. Use when asked to watch/babysit CI or PR checks.
disable-model-invocation: true
---

# Babysit CI

Reactively monitor CI on a GitHub PR (default repo: `stackrox/stackrox`) until every check
is terminal, triaging failures as they land instead of waiting for the full run to finish.

## Check types

`gh pr checks <PR-number> -R stackrox/stackrox` lists both kinds:
- **GitHub Actions** — plain names (e.g. `go (GOTAGS="")`, `style-check`,
  `local-roxctl-tests`). They start automatically, but **SKIP on draft PRs**.
- **Prow jobs** — names prefixed `ci/prow/` (e.g. `ci/prow/gke-qa-e2e-tests`). They run
  only when explicitly triggered with a `/test` comment, and they **DO work on draft PRs**.

## Workflow

1. **Trigger tests (only if the user asks, or if no tests are running yet).** Post ALL
   Prow `/test <job>` commands as a SINGLE PR comment — never one comment per job.
   Typical relevant jobs: `gke-qa-e2e-tests`, `ocp-4-12-qa-e2e-tests`,
   `gke-version-compatibility-tests`, `gke-nongroovy-compatibility-tests`.

2. **Arm a background watcher.** Run the bundled `watch-checks.sh` (in this skill's
   directory) via Bash with `run_in_background: true`:

   ```bash
   .claude/skills/babysit-ci/watch-checks.sh <PR-number> [--only ci/prow] [--interval 90] [--timeout 360]
   ```

   Its first poll establishes the baseline (possibly empty when no checks run yet); it
   exits as soon as a check newly reaches a terminal state, printing only the new
   result(s). The snapshot is persisted to a state file, so re-arming the watcher after
   a notification does not miss results that land in between. If the harness offers a
   `Monitor` tool with until-loop support, prefer that; the script is the portable
   fallback and reference implementation.

3. **React to each watcher notification immediately.** Read the background task's output
   file; it prints only the newly terminal check(s) and their pass/fail state. Then re-arm
   the watcher (run the script again in the background) so later results keep arriving.
   - **Pass** — report briefly and re-arm.
   - **Fail** — do NOT wait for the remaining checks; start triage right away and re-arm
     the watcher in parallel:
     - `ci/prow/` failure: launch the `stackrox-ci-failure-investigator` agent with the PR
       number and the failing job name; it downloads artifacts and does root-cause analysis.
     - GitHub Actions failure: fetch the log with `gh run view <run-id> --log-failed` and
       analyze it yourself.

4. **Finish.** When every check is terminal (or the user calls it off), give a summary
   table of all checks and their outcomes, and flag anything that failed.

## Cautions

- Never use `gh pr view --json ...` in scripts — it can fail on tokens missing the
  `read:org` scope. Use `gh pr checks` or the REST API (`gh api repos/.../pulls/N`).
- `gh pr checks` exits non-zero when any check has failed; scripts must not treat that
  exit code as a fatal error.
