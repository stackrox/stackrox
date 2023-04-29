#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"
# # shellcheck source=../../scripts/ci/lib.sh
# source "$ROOT/scripts/ci/lib.sh"

define_build_matrix() {

    read -r -d '' matrix <<- _EO_MATRIX_ || true
    {
        "pre_build_cli": { "name": ["development"] },
        "pre_build_go_binaries": { "name": ["development", "race-condition-debug"] },
        "build_and_push_main": { "name": ["STACKROX_BRANDING", "RHACS_BRANDING", "race-condition-debug"] }
    }
_EO_MATRIX_

    info "Base build matrix is:"
    jq <<< "$matrix"

    if true; then
        matrix="$(jq '.pre_build_cli.name += ["prerelease"]' <<< "$matrix")"
        matrix="$(jq '.pre_build_go_binaries.name += ["prerelease"]' <<< "$matrix")"
        matrix="$(jq '.build_and_push_main.name += ["prerelease"]' <<< "$matrix")"

        info "Build matrix after prerelease addition:"
        jq <<< "$matrix"
    fi

    condensed="$(jq -c <<< "$matrix")"
    echo "matrix=$condensed" >> "$GITHUB_OUTPUT"
}
