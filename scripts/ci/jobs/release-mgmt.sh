#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

release_mgmt() {
    info "Release management steps"

    local tag
    tag="$(make --quiet tag)"

    slack_build_notice "$tag"

    if is_release_version "$tag"; then
        mark_collector_release "$tag"
    fi

    local release_issues=()

    if is_RC_version "${tag}" && ! check_docs "${tag}"; then
        release_issues+=("docs/ is not valid for a release.")
    fi

    if is_RC_version "${tag}" && ! check_scanner_and_collector_versions; then
        release_issues+=("SCANNER_VERSION and COLLECTOR_VERSION need to also be release.")
    fi

    if [[ "${#release_issues[@]}" != "0" ]]; then
        info "ERROR: Issues were found:"
        for issue in "${release_issues[@]}"; do
            echo -e "\t$issue"
        done
        exit 1
    fi
}

slack_build_notice() {
    info "Slack a build notice"

    if [[ "$#" -lt 1 ]]; then
        die "missing arg. usage: slack_build_notice <tag>"
    fi

    local tag="$1"

    [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]] || {
        info "Skipping step as this is not a release or RC build"
        return 0
    }

    local release
    release="$(get_release_stream "$tag")"

    local webhook_url
    if ! is_release_test_stream "$tag"; then
        # send to #eng-release
        webhook_url="${RELEASE_WORKFLOW_NOTIFY_WEBHOOK}"
    else
        # send to #slack-test when testing the release process
        webhook_url="${SLACK_MAIN_WEBHOOK}"
    fi

    jq -n \
    --arg release "$release" \
    --arg tag "$tag" \
    '{"text": "Prow build for tag `\($tag)` started! Check the status of the build under the following URL: https://prow.ci.openshift.org/?repo=stackrox%2Fstackrox&job=*release-\($release).x*"}' \
| curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url"
}

release_mgmt "$@"
