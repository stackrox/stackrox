#!/usr/bin/env bash

# A collection of GCP related reusable bash functions for CI

set -euo pipefail

setup_gcp() {
    info "Setting up GCP auth and config"

    local service_account
    if [[ -n "${GCP_SERVICE_ACCOUNT_STACKROX_CI:-}" ]]; then
        service_account="${GCP_SERVICE_ACCOUNT_STACKROX_CI}"
    elif [[ -n "${GCLOUD_SERVICE_ACCOUNT_CI_ROX:-}" ]]; then
        # ROX-19985 stackrox/jenkins-plugin relies on this env name. This can be
        # removed once that issue is closed.
        service_account="${GCLOUD_SERVICE_ACCOUNT_CI_ROX}"
    else
        die "Support is missing for this environment"
    fi

    require_executable "gcloud"

    if [[ "$(gcloud config get-value core/project 2>/dev/null)" == "acs-san-stackroxci" ]]; then
        echo "Current project is already set to acs-san-stackroxci. Assuming configuration already applied."
        return
    fi
    gcloud auth activate-service-account --key-file <(echo "$service_account")
    gcloud auth list
    gcloud config set project acs-san-stackroxci
    gcloud config set compute/region us-central1
    gcloud config unset compute/zone
    gcloud config set core/disable_prompts True

    # Some tools require a credential file for API calls e.g. prometheus-metric-parser
    touch /tmp/gcp.json
    chmod 0600 /tmp/gcp.json
    echo "$service_account" >/tmp/gcp.json
    ci_export GOOGLE_APPLICATION_CREDENTIALS /tmp/gcp.json
}
