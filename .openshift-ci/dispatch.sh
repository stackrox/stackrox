#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

shopt -s nullglob
for cred in /tmp/secret/**/[A-Z]*; do
    export "$(basename "$cred")"="$(cat "$cred")"
done

openshift_ci_mods

function hold() {
    while [[ -e /tmp/hold ]]; do
        info "Holding this job for debug"
        sleep 60
    done
}
trap hold EXIT

if [[ "$#" -lt 1 ]]; then
    die "usage: dispatch <ci-job> [<...other parameters...>]"
fi

ci_job="$1"
shift
ci_export CI_JOB_NAME "$ci_job"

gate_job "$ci_job"

case "$ci_job" in
    gke-qa-e2e-tests|gke-nongroovy-e2e-tests|gke-upgrade-tests|gke-ui-e2e-tests)
        openshift_ci_e2e_mods
        ;;
esac

case "$ci_job" in
    policy-checks)
        "$ROOT/scripts/ci/jobs/check-policy-files.sh"
        ;;
    style-checks)
        make style
        ;;
    generated-checks)
        "$ROOT/scripts/ci/jobs/check-generated.sh"
        ;;
    todo-checks)
        "$ROOT/scripts/ci/jobs/check-todos.sh"
        ;;
    pr-fixes-checks)
        "$ROOT/scripts/ci/jobs/check-pr-fixes.sh"
        ;;
    mitre-bundles-checks)
        "$ROOT/scripts/ci/jobs/check-mitre-bundles.sh"
        ;;
    go-unit-tests-release)
        GOTAGS=release "$ROOT/scripts/ci/jobs/go-unit-tests.sh"
        ;;
    go-unit-tests)
        GOTAGS='' "$ROOT/scripts/ci/jobs/go-unit-tests.sh"
        ;;
    go-postgres-tests)
        GOTAGS='' "$ROOT/scripts/ci/jobs/go-postgres-tests.sh"
        ;;
    integration-unit-tests)
        "$ROOT/scripts/ci/jobs/integration-unit-tests.sh"
        ;;
    shell-unit-tests)
        "$ROOT/scripts/ci/jobs/shell-unit-tests.sh"
        ;;
    ui-unit-tests)
        "$ROOT/scripts/ci/jobs/ui-unit-tests.sh"
        ;;
    push-images)
        "$ROOT/scripts/ci/jobs/push-images.sh" "$@"
        ;;
    release-mgmt)
        "$ROOT/scripts/ci/jobs/release-mgmt.sh" "$@"
        ;;
    gke-qa-e2e-tests)
        "$ROOT/.openshift-ci/gke_qa_e2e_test.py"
        ;;
    gke-nongroovy-e2e-tests)
        "$ROOT/.openshift-ci/gke_nongroovy_e2e_test.py"
        ;;
    openshift-4-qa-e2e-tests)
        "$ROOT/.openshift-ci/openshift_4_qa_e2e_test.py"
        ;;
    gke-upgrade-tests)
        "$ROOT/.openshift-ci/gke_upgrade_test.py"
        ;;
    gke-ui-e2e-tests)
        "$ROOT/.openshift-ci/gke_ui_e2e_test.py"
        ;;
    *)
        # For ease of initial integration this function does not fail when the
        # job is unknown.
        info "nothing to see here: ${ci_job}"
        exit 0
esac
