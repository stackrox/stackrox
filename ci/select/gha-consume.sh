#!/usr/bin/env bash
set -euo pipefail
# Parse outputs outside workflow expressions so malformed JSON cannot block tests.
action=run
if [[ "${CI_SELECT_PLANNER_RESULT:-}" == success ]] &&
    jq -e -s --arg job "$1" '
        length == 1 and (.[0] |
            type == "object" and
            all(.[]; . == "run" or . == "skip") and
            .[$job] == "skip")
    ' <<<"${CI_SELECT_DECISIONS:-}" >/dev/null 2>&1; then
    action=skip
fi
printf 'action=%s\n' "$action" >>"$GITHUB_OUTPUT"
printf '%s: applied=%s\n' "$1" "$action"
