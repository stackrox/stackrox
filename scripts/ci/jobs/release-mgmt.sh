#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

release_mgmt() {
    info "Release management steps"

    local release_issues=()

    local tag
    tag="$(make --quiet tag)"

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

release_mgmt "$@"
