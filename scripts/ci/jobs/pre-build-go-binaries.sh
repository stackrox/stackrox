#!/usr/bin/env bash

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"

set -euo pipefail

source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

pre_build_go_binaries() {
    # TODO(RS-509) - PR labels cannot be queried on the private rox-openshift-ci-mirror
    # if pr_has_label "ci-release-build"; then
    #     ci_export GOTAGS release
    # fi

    # if pr_has_label "ci-race-tests"; then
    #     RACE=true make main-build-nodeps
    # else
    #     make main-build-nodeps
    # fi

    make main-build-nodeps

    make swagger-docs
}

pre_build_go_binaries
