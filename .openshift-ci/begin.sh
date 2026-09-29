#!/usr/bin/env bash

# The initial script executed for openshift/release CI jobs.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"

set -euo pipefail

info "Start of CI handling"

# Skip expensive setup for non-UI jobs when the PR only touches ui/ files.
if [[ "${JOB_NAME:-}" =~ nongroovy|upgrade ]] && changes_limited_to "ui/"; then
    info "Skipping begin: UI-only PR, non-UI job ${JOB_NAME}"
    exit 0
fi

openshift_ci_mods
openshift_ci_import_creds

setup_gcp

# Pre-flight DNS check (mitigation for DPTP-5138)
info "Pre-flight DNS health check (DPTP-5138 mitigation)"

_check_dns() {
    local domain="$1"
    getent hosts "$domain" >/dev/null 2>&1
}

dns_check_failed=false
for domain in oauth2.googleapis.com compute.googleapis.com storage.googleapis.com; do
    if retry 5 true _check_dns "$domain"; then
        info "✓ DNS OK for $domain"
    else
        warn "⚠️  DNS resolution FAILED for $domain after retries"
        dns_check_failed=true
    fi
done

if [[ "$dns_check_failed" == "true" ]]; then
    local error_msg="DNS is broken after retries (likely persistent node-resolver issue - see https://redhat.atlassian.net/browse/DPTP-5138). Failing fast to avoid wasted cluster creation time."

    # Generate structured JUnit report for better CI dashboard visibility
    if [[ -n "${ARTIFACT_DIR:-}" ]] && command -v save_junit_failure >/dev/null 2>&1; then
        save_junit_failure "DNS_Preflight_Check" "DNS resolution failed" "$error_msg"
    fi

    die "$error_msg"
fi

set_ci_shared_export started_at "$(date -u +%s)"

if [[ -z "${SHARED_DIR:-}" ]]; then
    echo "ERROR: There is no SHARED_DIR for step env sharing"
    exit 0 # not fatal but worth highlighting
fi

if [[ "${JOB_NAME:-}" =~ -ocp- ]]; then
    info "Setting worker node type and count for OCP 4 jobs"
    set_ci_shared_export WORKER_NODE_COUNT 2
    if [[ "${JOB_NAME:-}" =~ vm-scanning ]]; then
        # Selecting nodes with KVM support
        set_ci_shared_export WORKER_NODE_TYPE n2-standard-8
    else
        set_ci_shared_export WORKER_NODE_TYPE e2-standard-8
    fi
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
