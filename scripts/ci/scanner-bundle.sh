#!/usr/bin/env bash

# shellcheck source=./nightly.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/nightly.sh"

# Return the published version of the checked-in Scanner CI bundle.
scanner_v4_ci_bundle_version() {
    local root="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
    local bundle="$root/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
    if [[ ! -s "$bundle" ]]; then
        echo "Missing or empty Scanner CI bundle: $bundle" >&2
        return 1
    fi
    local digest
    digest="$(sha256sum "$bundle")" || return 1
    printf 'ci-minimal-%s\n' "${digest%% *}"
}

# Explicit full-feed testing and nightlies retain the deployed version defaults.
scanner_v4_use_full_bundle() {
    [[ "${SCANNER_V4_FULL_BUNDLE:-false}" == true ]] && return 0
    is_nightly_run && return 0
    if declare -F is_in_PR_context >/dev/null && is_in_PR_context; then
        pr_has_label ci-scanner-full-bundle && return 0
    fi
    return 1
}
