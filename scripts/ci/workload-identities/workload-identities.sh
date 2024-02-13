#!/usr/bin/env bash

# Workload identity resources for Central.

# shellcheck source=../../../scripts/ci/gcp.sh
source "$SCRIPTS_ROOT/scripts/ci/gcp.sh"

set -euo pipefail

setup_workload_identities() {
    setup_gcp_workload_identities
}

cleanup_workload_identities() {
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
        "$SCRIPTS_ROOT/scripts/ci/workload-identities/sts-config.json" | base64 | tr -d "\n")
    CREDENTIALS=${sts_config} envsubst < \
        "$SCRIPTS_ROOT/scripts/ci/workload-identities/gcp-cloud-credentials.yaml" | kubectl apply -f -
}

cleanup_gcp_workload_identities() {
    info "Cleaning up GCP workload identities."

    setup_gcp
    setup_gcp_variables

    gcloud iam service-accounts remove-iam-policy-binding "${service_account}" \
        --member="principal://iam.googleapis.com/projects/${project}/locations/global/workloadIdentityPools/${cluster}/subject/${subject_central}" \
        --role=roles/iam.workloadIdentityUser
}
