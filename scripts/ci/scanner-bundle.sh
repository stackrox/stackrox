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

_scanner_v4_ci_helm_values() {
    local values_file="$1"
    local disabled explicit
    disabled="$(yq eval '.scannerV4.disable // false' "$values_file")" || return 1
    explicit="$(yq eval '(.customize."scanner-v4-matcher".envVars // {}) | (has("SCANNER_V4_MATCHER_VULNERABILITIES_URL") or has("SCANNER_V4_MATCHER_VULNERABILITY_VERSION"))' "$values_file")" || return 1
    if scanner_v4_use_full_bundle || [[ "${ROX_SCANNER_V4:-}" == false || "$disabled" == true || "$explicit" == true ]]; then
        cat "$values_file"
        return
    fi
    local root version
    root="$(cd "$(dirname "$values_file")/../.." && pwd)"
    version="$(scanner_v4_ci_bundle_version "$root")" || return 1
    echo "Scanner V4 bundle: $version via Central" >&2
    SCANNER_INSTALL_BUNDLE_VERSION="$version" yq eval '.customize."scanner-v4-matcher".envVars.SCANNER_V4_MATCHER_VULNERABILITY_VERSION = strenv(SCANNER_INSTALL_BUNDLE_VERSION)' "$values_file"
}

# Static local values carry a pin, but scripted full-feed runs use the image stream.
_scanner_v4_local_helm_values() {
    local values_file="$1"
    if scanner_v4_use_full_bundle || [[ "${ROX_SCANNER_V4:-}" == false ]]; then
        yq eval 'del(.customize."scanner-v4-matcher".envVars.SCANNER_V4_MATCHER_VULNERABILITY_VERSION)' "$values_file"
    else
        cat "$values_file"
    fi
}

_scanner_v4_install_bundle_values() {
    if scanner_v4_use_full_bundle; then
        echo "Scanner V4 bundle: production (deployed version default)" >&2
        return 0
    fi
    local version
    version="$(scanner_v4_ci_bundle_version "$TEST_ROOT")" || return 1
    echo "Scanner V4 bundle: $version via Central" >&2
    printf 'customize:\n  scanner-v4-matcher:\n    envVars:\n      SCANNER_V4_MATCHER_VULNERABILITY_VERSION: %s\n' "$version"
}

_configure_roxie_ci_vuln_bundle() {
    local config_file="$1"
    [[ "${CI:-}" == "true" ]] || return 0
    if scanner_v4_use_full_bundle; then
        echo "Scanner V4 bundle: production default unless explicitly overridden in Roxie config" >&2
        return 0
    fi

    local scanner_v4_component
    if ! scanner_v4_component="$(yq eval '.central.spec.scannerV4.scannerComponent // ""' "$config_file")"; then
        return 1
    fi
    [[ "$scanner_v4_component" != "Disabled" ]] || return 0

    local env_var_count
    if ! env_var_count="$(yq eval '[((.central.spec.customize.envVars // [])[]) | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL" or .name == "SCANNER_V4_MATCHER_VULNERABILITY_VERSION")] | length' "$config_file")"; then
        return 1
    fi
    if [[ "$env_var_count" != "0" ]]; then
        echo "Scanner V4 bundle: explicit override $(yq eval '.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL" or .name == "SCANNER_V4_MATCHER_VULNERABILITY_VERSION") | .value' "$config_file")" >&2
        return 0
    fi

    local version
    version="$(scanner_v4_ci_bundle_version "$TEST_ROOT")" || return 1
    echo "Scanner V4 bundle: $version via Central" >&2
    set_custom_env "$config_file" central SCANNER_V4_MATCHER_VULNERABILITY_VERSION "$version"
}
