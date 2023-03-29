#!/usr/bin/env bash

# Common functions for deploying a cluster for QA tests

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

set -euo pipefail

deploy_default_psp() {
    info "Deploy Default PSP for stackrox namespace"
    if [[ "$POD_SECURITY_POLICIES" != "false" ]]; then
        "${ROOT}/scripts/ci/create-default-psp.sh"
    else
        info "POD_SECURITY_POLICIES is false, skip Deploy Default PSP for stackrox namespace"
    fi
}

get_ECR_docker_pull_password() {
    info "Get AWS ECR Docker Pull Password"

    aws --version
    local pass
    pass="$(aws --region="${AWS_ECR_REGISTRY_REGION}" ecr get-login-password)"
    ci_export AWS_ECR_DOCKER_PULL_PASSWORD "${pass}"
}

deploy_clair_v4() {
    info "Deploy Clair v4"

    ci_export ROX_CLAIR_V4_SCANNING "${ROX_CLAIR_V4_SCANNING:-true}"
    ci_export CLAIR_V4_ENDPOINT "http://clairv4.qa-clairv4:8080"
    "$ROOT/scripts/ci/clairv4/deploy.sh" qa-clairv4
}
