#!/usr/bin/env bash

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
