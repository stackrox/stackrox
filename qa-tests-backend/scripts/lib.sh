#!/usr/bin/env bash

# Common functions for deploying a cluster for QA tests

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

set -euo pipefail

deploy_default_psp() {
    info "Deploy Default PSP for stackrox namespace"
    "${ROOT}/scripts/ci/create-default-psp.sh"
}

deploy_webhook_server() {
    info "Deploy Webhook server"

    local certs_dir
    certs_dir="$(mktemp -d)"
    "${ROOT}/scripts/ci/create-webhookserver.sh" "${certs_dir}"
    ci_export GENERIC_WEBHOOK_SERVER_CA_CONTENTS "$(cat "${certs_dir}/ca.crt")"
}

get_ECR_docker_pull_password() {
    info "Get AWS ECR Docker Pull Password"

    if [[ "${SKIP_ECR_TESTS:-}" != "true" ]]; then
        aws --version
        local pass
        pass="$(aws --region="${AWS_ECR_REGISTRY_REGION}" ecr get-login-password)"
        ci_export AWS_ECR_DOCKER_PULL_PASSWORD "${pass}"
    else
        info "ECR tests are skipped with ${SKIP_ECR_TESTS:-}"
        ci_export AWS_ECR_DOCKER_PULL_PASSWORD "skipped"
    fi
}
