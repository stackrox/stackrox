#!/usr/bin/env bash

# The entrypoint for CI defined in https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox
# Imports secrets to env vars, gates the job based on context, changed files and PR labels and ultimately
# hands off to the test/build script in *scripts/ci/jobs*.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

shopt -s nullglob
for cred in /tmp/secret/**/[A-Z]*; do
    export "$(basename "$cred")"="$(cat "$cred")"
done

info "Current Status:"
"$ROOT/status.sh" || true

openshift_ci_mods
handle_nightly_runs

info "Status after mods:"
"$ROOT/status.sh" || true

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

export PYTHONPATH="${PYTHONPATH:-}:.openshift-ci"

if ! [[ "$ci_job" =~ [a-z-]+ ]]; then
    # don't exec possibly untrusted scripts
    die "untrusted job: $ci_job"
fi

if [[ -f "$ROOT/scripts/ci/jobs/${ci_job}.sh" ]]; then
    job_script="$ROOT/scripts/ci/jobs/${ci_job}.sh"
elif [[ -f "$ROOT/scripts/ci/jobs/${ci_job//-/_}.py" ]]; then
    job_script="$ROOT/scripts/ci/jobs/${ci_job//-/_}.py"
else
    # For ease of initial integration this function does not fail when the
    # job is unknown.
    info "nothing to see here: ${ci_job}"
    exit 0
fi

"${job_script}" "$@"
