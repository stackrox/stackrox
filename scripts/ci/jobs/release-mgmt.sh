#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/gcp.sh"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

release_mgmt() {
    info "Release management steps"

    local tag
    tag="$(make --quiet tag)"

    if is_release_version "$tag"; then
        push_release "$tag"
        mark_collector_release "$tag"
    elif is_RC_version "$tag"; then
        local pre_release_warnings=()

        if ! check_docs "${tag}"; then
            pre_release_warnings+=("docs/ is not valid for a release.")
        fi

        if ! check_scanner_and_collector_versions; then
            pre_release_warnings+=("SCANNER_VERSION and COLLECTOR_VERSION need to also be release.")
        fi

        if [[ "${#pre_release_warnings[@]}" != "0" ]]; then
            info "ERROR: Issues were found:"
            for issue in "${pre_release_warnings[@]}"; do
                echo -e "\t$issue"
            done
            exit 1
        fi
    fi
}

release_mgmt "$@"
