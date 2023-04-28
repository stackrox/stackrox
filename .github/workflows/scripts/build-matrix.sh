#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test utility functions

# TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# # shellcheck source=../../scripts/lib.sh
# source "$TEST_ROOT/scripts/lib.sh"
# # shellcheck source=../../scripts/ci/lib.sh
# source "$TEST_ROOT/scripts/ci/lib.sh"

define_build_matrix() {

    read -r -d '' matrix <<- _EO_MATRIX_ || true
    {
        "pre_build_cli": {
            "include": [
                {"name": "development", "release": false, "artifact": "cli-build-development"},
                {"name": "prerelease", "release": true, "artifact": "cli-build-prerelease"}
            ]
        },
        "pre_build_go_binaries": {
            "include": [
                {"name": "development", "release": false, "artifact": "go-binaries-build-development"},
                {"name": "race condition debug", "release": false, "artifact": "go-binaries-build-rcd"},
                {"name": "prerelease", "release": true, "artifact": "go-binaries-build-prerelease"}
            ]
        },
        "build_and_push_main": {
            "include": [
                {"name": "stackrox branding", "branding": "STACKROX_BRANDING", "cli-artifact": "cli-build-development", "go-binaries-artifact": "go-binaries-build-development"},
                {"name": "rhacs branding", "branding": "RHACS_BRANDING", "cli-artifact": "cli-build-development", "go-binaries-artifact": "go-binaries-build-development"},
                {"name": "race condition debug", "branding": "STACKROX_BRANDING", "cli-artifact": "cli-build-development", "go-binaries-artifact": "go-binaries-build-rcd"},
                {"name": "prerelease", "branding": "RHACS_BRANDING", "cli-artifact": "cli-build-prerelease", "go-binaries-artifact": "go-binaries-build-prerelease"}
            ]
        }
    }
_EO_MATRIX_

    jq <<< "$matrix"

    condensed="$(jq -c <<< "$matrix")"

    echo "matrix=$condensed" >> "$GITHUB_OUTPUT"
}
