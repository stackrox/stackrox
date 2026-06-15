#!/usr/bin/env bash

# Workload identity resources for Central.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"
# shellcheck source=../../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"

set -euo pipefail

setup_workload_identities() {
    if [[ "${SETUP_WORKLOAD_IDENTITIES:-false}" == "false" ]]; then
        info "Skipping the workload identity setup."
        return 0
    fi
    setup_gcp_workload_identities
}

cleanup_workload_identities() {
    if [[ "${SETUP_WORKLOAD_IDENTITIES:-false}" == "false" ]]; then
        info "Skipping the workload identity cleanup."
        return 0
    fi
    cleanup_gcp_workload_identities
}

setup_gcp_variables() {
    cluster=$(kubectl config view --minify -o jsonpath="{.clusters[].name}")
    service_account="${GCP_SERVICE_ACCOUNT_EMAIL_STACKROX_CI_WORKLOAD_IDENTITY}"
    project="${GCP_PROJECT_NUMBER_WORKLOAD_IDENTITY}"
    subject_central="system:serviceaccount:stackrox:central"
    iam_principal="principal://iam.googleapis.com/projects/${project}/locations/global/workloadIdentityPools/${cluster}/subject/${subject_central}"
}

setup_gcp_workload_identities() {
    info "Setting up GCP workload identities."

    setup_gcp
    setup_gcp_variables

    local creation_label="created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

    # Connect the stackrox ci service account to the workload identity of Central.
    # The condition acts as a creation timestamp label for later pruning of stale bindings.
    retry 5 true \
        gcloud iam service-accounts add-iam-policy-binding "${service_account}" \
        --member="${iam_principal}" \
        --role=roles/iam.workloadIdentityUser \
        --condition="title=${creation_label},expression=true"

    # Apply STS configuration.
    local -r sts_config=$(PROJECT=${project} CLUSTER=${cluster} SERVICE_ACCOUNT=${service_account} envsubst < \
        "$ROOT/qa-tests-backend/scripts/workload-identities/sts-config.json" | base64 | tr -d "\n")
    CREDENTIALS=${sts_config} envsubst < \
        "$ROOT/qa-tests-backend/scripts/workload-identities/gcp-cloud-credentials.yaml" | kubectl apply -f -
}

cleanup_gcp_workload_identities() {
    info "Cleaning up GCP workload identities."

    setup_gcp
    setup_gcp_variables

    # We need to query the condition title because `remove-iam-policy-binding` requires an exact match including condition.
    local condition_title
    condition_title=$(gcloud iam service-accounts get-iam-policy "${service_account}" --format=json | \
        jq -r --arg iam_principal "${iam_principal}" \
        '.bindings[] | select(.role=="roles/iam.workloadIdentityUser") | select(.members[]? == $iam_principal) | .condition.title')

    retry 5 true \
        gcloud iam service-accounts remove-iam-policy-binding "${service_account}" \
        --member="${iam_principal}" \
        --role=roles/iam.workloadIdentityUser \
        --condition="title=${condition_title},expression=true"
}
