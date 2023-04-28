#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

check_policy_files() {
    info 'Ensure all JSON policies in "./pkg/defaults/policies/" are of latest version.'
    # shellcheck disable=SC2016
    info '(If this fails, run `policyutil` on failed policies and commit the result.)'

    make deps
    make policyutil
    policyutil upgrade -d pkg/defaults/policies/files -o /tmp/policies-in-standard-form --ensure-read-only mitre --ensure-read-only criteria
    diff pkg/defaults/policies/files /tmp/policies-in-standard-form > /tmp/policies-diff || true

    store_test_results /tmp/policies-diff policies-diff

    if [[ -s /tmp/policies-diff ]]; then
        echo 'error: Found policies that are not in standard form.' \
            'Check "policies-diff" for affected policies; use "policyutil" to fix them.'
        cat /tmp/policies-diff
        exit 1
    fi
}

check_policy_files
