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
