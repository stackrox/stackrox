#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

info "Start of CI handling"

openshift_ci_mods
openshift_ci_import_creds

create_job_record "${JOB_NAME:-missing}" "prow"

if [[ -z "${SHARED_DIR:-}" ]]; then
    echo "ERROR: There is no SHARED_DIR for step env sharing"
    exit 0 # not fatal but worth highlighting
fi

if [[ "${JOB_NAME:-}" =~ -ocp- ]]; then
    info "Setting worker node type and count for OCP 4 jobs"
    set_ci_shared_export WORKER_NODE_COUNT 2
    set_ci_shared_export WORKER_NODE_TYPE e2-standard-8
fi

if [[ "${JOB_NAME:-}" =~ -eks- ]]; then
    info "Provide access for the CI user to EKS"
    # shellcheck disable=SC2034
    AWS_ACCESS_KEY_ID="$(cat /tmp/vault/stackrox-stackrox-e2e-tests/AWS_ACCESS_KEY_ID)"
    # shellcheck disable=SC2034
    AWS_SECRET_ACCESS_KEY="$(cat /tmp/vault/stackrox-stackrox-e2e-tests/AWS_SECRET_ACCESS_KEY)"
    aws sts get-caller-identity | jq -r '.Arn'
    set_ci_shared_export USER_ARNS "$(aws sts get-caller-identity | jq -r '.Arn')"
fi
