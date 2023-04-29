#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_test_bin() {
    info "Making test-bin"

    if is_in_PR_context && pr_has_label "ci-skip-prow-jobs"; then
        # Can skip jobs by failing this initial build step (for jobs that run from: test-bin)
        die "ERROR: Skipping all prow (openshift/release) defined CI jobs"
    fi

    make cli-build upgrader
    install_built_roxctl_in_gopath
}

make_test_bin "$*"
