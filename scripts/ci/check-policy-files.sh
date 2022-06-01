#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"  

set -euo pipefail

# check-policy-files-up-to-date:
#    steps:
#      - checkout
#      - restore-go-mod-cache
#      - setup-go-build-env

# name: Ensure all JSON policies in "./pkg/defaults/policies/" are of latest version. (If this fails, run `policyutil` on failed policies and commit the result.)

check_policy_files() {
    info 'Ensure all JSON policies in "./pkg/defaults/policies/" are of latest version.'
    info '(If this fails, run `policyutil` on failed policies and commit the result.)'
    
    make deps
    make policyutil
    policyutil upgrade -d pkg/defaults/policies/files -o /tmp/policies-in-standard-form --ensure-read-only mitre --ensure-read-only criteria
    diff pkg/defaults/policies/files /tmp/policies-in-standard-form > /tmp/policies-diff
    if [[ -s /tmp/policies-diff ]]; then
        echo 'Found policies that are not in standard form. Check "policies-diff" for affected policies; use "policyutil" to fix them.'
        cat /tmp/policies-diff
        exit 1
    fi

    # - ci-artifacts/store:
    #    path: /tmp/policies-diff
    #    destination: policies-diff
    store_test_results /tmp/policies-diff policies-diff
}

