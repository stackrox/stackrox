#!/usr/bin/env bash

# A collection of GCP related reusable bash functions for CI

set -euo pipefail

setup_gcp() {
    info "Setting up GCP auth and config"

    local service_account
    if [[ -n "${GCP_SERVICE_ACCOUNT_STACKROX_CI:-}" ]]; then
        service_account="${GCP_SERVICE_ACCOUNT_STACKROX_CI}"
    else
        die "Support is missing for this environment"
    fi

    require_executable "gcloud"
    local gcp_credentials_file="/tmp/gcp.json"

    if [[ "$(gcloud config get-value core/project 2>/dev/null)" == "acs-san-stackroxci" ]]; then
        echo "Current project is already set to acs-san-stackroxci. Assuming configuration already applied."

        # In some cases we have "setup_gcp()" already finished, but exported environment variable is lost.
        # Here we want to ensure that after running "setup_gcp()" environment is properly set.
        ci_export GOOGLE_APPLICATION_CREDENTIALS "$gcp_credentials_file"

        return
    fi

    gcloud auth activate-service-account --key-file <(echo "$service_account")
    gcloud auth list
    gcloud config set project acs-san-stackroxci
    gcloud config set compute/region us-central1
    gcloud config unset compute/zone
    gcloud config set core/disable_prompts True

    # Some tools require a credential file for API calls e.g. prometheus-metric-parser
    touch "$gcp_credentials_file"
    chmod 0600 "$gcp_credentials_file"
    echo "$service_account" >"$gcp_credentials_file"
    ci_export GOOGLE_APPLICATION_CREDENTIALS "$gcp_credentials_file"
}
