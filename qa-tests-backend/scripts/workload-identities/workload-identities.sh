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
    service_account="stackrox-ci-workload-identity@acs-san-stackroxci.iam.gserviceaccount.com"
    project="280228816191" # acs-san-stackroxci
    subject_central="system:serviceaccount:stackrox:central"
}

setup_gcp_workload_identities() {
    info "Setting up GCP workload identities."

    setup_gcp
    setup_gcp_variables

    # Connect the stackrox ci service account to the workload identity of Central.
    gcloud iam service-accounts add-iam-policy-binding "${service_account}" \
        --member="principal://iam.googleapis.com/projects/${project}/locations/global/workloadIdentityPools/${cluster}/subject/${subject_central}" \
        --role=roles/iam.workloadIdentityUser

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

    gcloud iam service-accounts remove-iam-policy-binding "${service_account}" \
        --member="principal://iam.googleapis.com/projects/${project}/locations/global/workloadIdentityPools/${cluster}/subject/${subject_central}" \
        --role=roles/iam.workloadIdentityUser
}
