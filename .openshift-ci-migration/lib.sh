#!/usr/bin/env bash

set -euo pipefail

info() {
    echo "INFO: $(date): $*"
}

get_pr_details() {
    local pull_request
    local org
    local repo

    pull_request=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].pulls[0].number')
    org=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].org')
    repo=$(jq -r <<<"$CLONEREFS_OPTIONS" '.refs[0].repo')

    headers=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        headers+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi

    url="https://api.github.com/repos/${org}/${repo}/pulls/${pull_request}"
    curl -sS "${headers[@]}" "${url}"
}
