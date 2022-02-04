#!/usr/bin/env bash

set -euo pipefail

# A collection of GCP related reusable bash functions for CI

set +u
SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
set -u

source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

setup_gcp() {
    info "Setting up GCP auth and config"

    ensure_CI

    local service_account
    if is_CIRCLECI; then
        require_environment "GCLOUD_SERVICE_ACCOUNT_CIRCLECI_ROX"
        service_account="${GCLOUD_SERVICE_ACCOUNT_CIRCLECI_ROX}"
    elif is_OPENSHIFT_CI; then
        require_environment "GCLOUD_SERVICE_ACCOUNT_OPENSHIFT_CI_ROX"
        service_account="${GCLOUD_SERVICE_ACCOUNT_OPENSHIFT_CI_ROX}"
    else
        die "Support is missing for this CI environment"
    fi

    require_executable "gcloud"

    if [[ "$(gcloud config get-value core/project 2>/dev/null)" == "stackrox-ci" ]]; then
        echo "Current project is already set to stackrox-ci. Assuming configuration already applied."
        exit 0
    fi
    gcloud auth activate-service-account --key-file <(echo "$service_account")
    gcloud auth list
    gcloud config set project stackrox-ci
    gcloud config set compute/region us-central1
    gcloud config unset compute/zone
    gcloud config set core/disable_prompts True
}
