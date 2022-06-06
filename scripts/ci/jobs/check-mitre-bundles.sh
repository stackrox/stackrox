#!/bin/bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -xveou pipefail

if ! is_tagged; then
    echo "Not a tagged build, skipping MITRE ATT&CK bundle check"
    exit 0
fi

echo 'Skipping until ROX-8486 is resolved' && exit 0

# shellcheck disable=SC2016
echo 'Ensure MITRE ATT&CK bundle at "./pkg/mitre/files/mitre.json" is up-to-date. (If this fails, run `mitreutil fetch` and commit the result.)'

function check_mitre_attach_bundle_up_to_date() {
    make deps
    make mitre
    mitre fetch --domain enterprise --out /tmp/enterprise-mitre.json
    diff pkg/mitre/files/mitre.json /tmp/enterprise-mitre.json > /tmp/mitre-diff || true

    store_test_results /tmp/mitre-diff mitre-diff

    if [[ -s /tmp/mitre-diff ]]; then
        echo 'error: MITRE ATT&CK bundle at 'pkg/mitre/files/mitre.json' is not up-to-date. Check "mitre-diff" for more informtaion.'
        cat /tmp/mitre-diff
        exit 1
    fi
}

check_mitre_attach_bundle_up_to_date
