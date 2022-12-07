#!/usr/bin/env bash

set -euo pipefail

deploy_webhook_server() {
    info "Deploy Webhook server"

    local certs_dir
    certs_dir="${1:-$(mktemp -d)}"
    install_webhook_server "${certs_dir}"
    create_webhook_server_port_forward
    export_webhook_server_certs "${certs_dir}"
}

install_webhook_server() {
    certs_tmp_dir="$1"
    [[ -n "${certs_tmp_dir}" ]] || die "Usage: $0 <certs_dir>"
    [[ -d "${certs_tmp_dir}" ]] || mkdir "${certs_tmp_dir}"

    printstatus() {
        echo current resource status ...
        echo
        kubectl -n stackrox get all -l app=webhookserver
    }

    trap printstatus ERR

    gitroot="$(git rev-parse --show-toplevel)"
    [[ -n "${gitroot}" ]] || die "Could not determine git root"

    "${gitroot}/tests/scripts/setup-certs.sh" "${certs_tmp_dir}" webhookserver.stackrox "Webhook Server CA"
    cd "${gitroot}/webhookserver"
    mkdir -p chart/certs
    cp "${certs_tmp_dir}/tls.crt" "${certs_tmp_dir}/tls.key" chart/certs
    helm -n stackrox upgrade --install webhookserver chart/
}

create_webhook_server_port_forward() {
    sleep 5
    pod="$(kubectl -n stackrox get pod -l app=webhookserver -o name)"
    echo "Got pod ${pod}"
    [[ -n "${pod}" ]]
    kubectl -n stackrox wait --for=condition=ready "${pod}" --timeout=5m
    nohup kubectl -n stackrox port-forward "${pod}" 8080:8080 </dev/null > /dev/null 2>&1 &
    sleep 1
}

export_webhook_server_certs() {
    local certs_dir="$1"

    ci_export GENERIC_WEBHOOK_SERVER_CA_CONTENTS "$(cat "${certs_dir}/ca.crt")"
}
