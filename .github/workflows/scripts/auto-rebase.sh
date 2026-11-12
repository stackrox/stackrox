#!/usr/bin/env bash
#
# Rebases open PRs labeled `auto-rebase` onto their base branch.
#
set -euo pipefail

check_not_empty GITHUB_REPOSITORY

gh pr list \
    --search "label:auto-rebase" \
    --state open \
    --limit 1000 \
    --json number \
    --jq '.[].number' \
    | while IFS= read -r pr_number; do
          gh_summary "auto-rebasing #$pr_number"
          gh pr update-branch "$pr_number" --repo "$GITHUB_REPOSITORY" --rebase \
              || gh_log warning "auto-rebase failed: #$pr_number: skipping until next run"
      done
