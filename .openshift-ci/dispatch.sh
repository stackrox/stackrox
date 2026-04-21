#!/usr/bin/env bash

# The entrypoint for CI defined in https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox
# Imports secrets to env vars, gates the job based on context, changed files and PR labels and ultimately
# hands off to the test/build script in *scripts/ci/jobs*.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"

set -euo pipefail

if [[ -f "${SHARED_DIR:-}/shared_env" ]]; then
    # shellcheck disable=SC1091
    source "${SHARED_DIR:-}/shared_env"
fi

openshift_ci_mods
openshift_ci_import_creds
create_exit_trap

if [[ "$#" -lt 1 ]]; then
    die "usage: dispatch <ci-job> [<...other parameters...>]"
fi

ci_job="$1"
shift
ci_export CI_JOB_NAME "$ci_job"

case "$ci_job" in
    gke*qa-e2e-tests|gke*nongroovy-e2e-tests|gke*upgrade-tests|gke-ui-e2e-tests|\
    eks-qa-e2e-tests|osd*qa-e2e-tests|gke*sensor-integration-tests)
        openshift_ci_e2e_mods
        ;;
    *-operator-e2e-tests)
        operator_e2e_test_setup
        ;;
esac

case "$ci_job" in
    eks-qa-e2e-tests|osd*qa-e2e-tests|ocp*ui-e2e-tests)
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

if [[ "${JOB_NAME:-}" =~ -ocp- ]]; then
    info "Allow restricted SCC for all users and namespaces."
    oc create clusterrolebinding system:openshift:scc:restricted --clusterrole=system:openshift:scc:restricted --group=system:authenticated || true
fi

if [[ "$ci_job" =~ e2e|upgrade ]] && [[ "${USE_ROXIE_DEPLOY:-true}" != 'false' ]]; then
    if [[ ${USE_ROXIE_VERSION:-latest} == 'latest' ]]; then
        USE_ROXIE_VERSION="$(curl -s -H "Authorization: token ${RHACS_BOT_GITHUB_TOKEN}" \
          "https://api.github.com/repos/stackrox/roxie/releases/latest" | jq -r '.tag_name')"
    fi
    info "Installing roxie release ${USE_ROXIE_VERSION} ..."
    curl -o /tmp/roxie \
      -H "Authorization: token ${RHACS_BOT_GITHUB_TOKEN:-}" \
      -H "Accept:application/octet-stream"\
      "https://github.com/stackrox/roxie/releases/download/${USE_ROXIE_VERSION}/roxie-linux-amd64" \
      || true
    echo "$PATH"
    for bin_path in ${PATH//:/ }; do
        install -b /tmp/roxie "$bin_path"/roxie \
          || continue
        break
    done
    which roxie
    roxie version || true
fi

"${job_script}" "$@" &
job_pid="$!"

# An Openshift CI job is canceled and sent a SIGINT when for example a new
# commit is pushed to a PR.
forward_sigint() {
    echo "Dispatch is forwarding SIGINT to job"
    kill -SIGINT "${job_pid}"
    # Delay the default exit trap execution and process completion to allow job
    # SIGINT handlers to complete before ci-operator terminates.
    sleep 3
}
trap forward_sigint SIGINT

wait "${job_pid}"
