#!/usr/bin/env bash

# The entrypoint for CI defined in https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox
# Imports secrets to env vars, gates the job based on context, changed files and PR labels and ultimately
# hands off to the test/build script in *scripts/ci/jobs*.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/tests/e2e/lib.sh"

set -euo pipefail

openshift_ci_mods
openshift_ci_import_creds
create_exit_trap

if [[ "$#" -lt 1 ]]; then
    die "usage: dispatch <ci-job> [<...other parameters...>]"
fi

ci_job="$1"
shift
ci_export CI_JOB_NAME "$ci_job"

gate_job "$ci_job"

case "$ci_job" in
    gke*qa-e2e-tests|gke-nongroovy-e2e-tests|gke*upgrade-tests|gke-ui-e2e-tests|\
    eks-qa-e2e-tests|osd*qa-e2e-tests)
        openshift_ci_e2e_mods
        ;;
    openshift-*-operator-e2e-tests)
        operator_e2e_test_setup
        ;;
esac

case "$ci_job" in
    eks-qa-e2e-tests|osd*qa-e2e-tests)
        setup_automation_flavor_e2e_cluster "$ci_job"
        ;;
esac

if [[ "$ci_job" =~ e2e|upgrade ]]; then
    handle_nightly_binary_version_mismatch
fi

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
    die "ERROR: There is no job script for $ci_job"
fi

"${job_script}" "$@" &
job_pid="$!"

forward_sigint() {
    echo "Dispatch is forwarding SIGINT to job"
    kill -SIGINT "${job_pid}"
}
trap forward_sigint SIGINT

wait "${job_pid}"
