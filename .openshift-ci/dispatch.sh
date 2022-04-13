#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

for cred in /tmp/secret/**/[A-Z]*; do
    export "$(basename "$cred")"="$(cat "$cred")"
done

openshift_ci_mods

if pr_has_label "delay-tests"; then
    function hold() {
        info "Holding on for debug"
        sleep 3600
    }
    trap hold EXIT
fi

if [[ "$#" -lt 1 ]]; then
    die "usage: dispatch <ci-job> [<...other parameters...>]"
fi

ci_job="$1"
shift

case "$ci_job" in
    gke-upgrade-tests)
        "$ROOT/.openshift-ci/gke_upgrade_test.py"
        ;;
    *)
        # For ease of initial integration this function does not fail when the
        # job is unknown.
        info "nothing to see here"
        exit 0
esac
