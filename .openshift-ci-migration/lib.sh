#!/usr/bin/env bash

set -euo pipefail

info() {
    echo "INFO: $(date): $*"
}

die() {
    echo >&2 "$@"
    exit 1
}

is_CI() {
    [[ "${CI:-}" == "true" ]]
}

is_CIRCLECI() {
    [[ "${CIRCLECI:-}" == "true" ]]
}

is_OPENSHIFT_CI() {
    [[ "${OPENSHIFT_CI:-}" == "true" ]]
}

require_environment() {
    if [[ "$#" -lt 1 ]]; then
        die "usage: require_environment NAME [reason]"
    fi

    (
        set +u
        if [[ -z "$(eval echo "\$$1")" ]]; then
            varname="$1"
            shift
            message="missing \"$varname\" environment variable"
            if [[ "$#" -gt 0 ]]; then
                message="$message: $*"
            fi
            die "$message"
        fi
    )
}

require_executable() {
    if [[ "$#" -lt 1 ]]; then
        die "usage: require_executable NAME [reason]"
    fi

    if ! command -v "$1" >/dev/null 2>&1; then
        varname="$1"
        shift
        message="missing \"$varname\" executable"
        if [[ "$#" -gt 0 ]]; then
            message="$message: $*"
        fi
        die "$message"
    fi
}

pr_has_label() {
    if [[ -z "${1:-}" ]]; then
        die "usage: pr_has_label <expected label>"
    fi

    require_environment "GITHUB_TOKEN"

    local expected_label="$1"
    get_pr_details | jq '([.labels | .[].name]  // []) | .[]' -r | grep -qx "${expected_label}"
}

get_pr_details() {
    require_environment "GITHUB_TOKEN"

    local pull_request
    local org
    local repo

    if is_CIRCLECI; then
        [ -n "${CIRCLE_PULL_REQUEST}" ] || { echo "Not on a PR, ignoring label overrides"; exit 3; }
        [ -n "${CIRCLE_PROJECT_USERNAME}" ] || { echo "CIRCLE_PROJECT_USERNAME not found" ; exit 2; }
        [ -n "${CIRCLE_PROJECT_REPONAME}" ] || { echo "CIRCLE_PROJECT_REPONAME not found" ; exit 2; }
        pull_request="${CIRCLE_PULL_REQUEST}"
        org="${CIRCLE_PROJECT_USERNAME}"
        repo="${CIRCLE_PROJECT_REPONAME}"
    elif is_OPENSHIFT_CI; then
        pull_request=$(jq -r <<<"$JOB_SPEC" '.refs.pulls[0].number')
        org=$(jq -r <<<"$JOB_SPEC" '.refs.org')
        repo=$(jq -r <<<"$JOB_SPEC" '.refs.repo')
    else
        die "not supported"
    fi

    url="https://api.github.com/repos/${org}/${repo}/pulls/${pull_request}"
    curl -sS -H "Authorization: token ${GITHUB_TOKEN}" "${url}"
}
