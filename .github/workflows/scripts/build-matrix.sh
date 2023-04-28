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
        "pre_build_cli": {
            "include": [
                {"name": "development", "artifact": "cli-build-development"}
            ]
        },
        "pre_build_go_binaries": {
            "include": [
                {"name": "development", "artifact": "go-binaries-build-development"},
                {"name": "race condition debug", "artifact": "go-binaries-build-rcd"}
            ]
        },
        "build_and_push_main": {
            "include": [
                {"name": "stackrox branding", "branding": "STACKROX_BRANDING", "cli-artifact": "cli-build-development", "go-binaries-artifact": "go-binaries-build-development"},
                {"name": "rhacs branding", "branding": "RHACS_BRANDING", "cli-artifact": "cli-build-development", "go-binaries-artifact": "go-binaries-build-development"},
                {"name": "race condition debug", "branding": "STACKROX_BRANDING", "cli-artifact": "cli-build-development", "go-binaries-artifact": "go-binaries-build-rcd"}
            ]
        }
    }
_EO_MATRIX_

    info "Base build matrix is:"
    jq <<< "$matrix"

    if true; then
        matrix="$(jq '.pre_build_cli.include += [{"name": "prerelease", "artifact": "cli-build-prerelease"}]' <<< "$matrix")"
        matrix="$(jq '.pre_build_go_binaries.include += [{"name": "prerelease", "artifact": "go-binaries-build-prerelease"}]' <<< "$matrix")"
        matrix="$(jq '.build_and_push_main.include += [{"name": "prerelease", "branding": "RHACS_BRANDING", "cli-artifact": "cli-build-prerelease", "go-binaries-artifact": "go-binaries-build-prerelease"}]' <<< "$matrix")"

        info "Build matrix after prerelease addition:"
        jq <<< "$matrix"
    fi

    condensed="$(jq -c <<< "$matrix")"
    echo "matrix=$condensed" >> "$GITHUB_OUTPUT"
}
