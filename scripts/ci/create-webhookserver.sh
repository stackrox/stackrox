#!/usr/bin/env bash

set -euo pipefail

# shellcheck disable=SC2120
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

    gitroot="$(git rev-parse --show-toplevel)"
    [[ -n "${gitroot}" ]] || die "Could not determine git root"

    "${gitroot}/tests/scripts/setup-certs.sh" "${certs_tmp_dir}" webhookserver.stackrox "Webhook Server CA"
    pushd "${gitroot}/webhookserver"
    mkdir -p chart/certs
    cp "${certs_tmp_dir}/tls.crt" "${certs_tmp_dir}/tls.key" chart/certs
    helm -n stackrox upgrade --install webhookserver chart/
    popd
}

create_webhook_server_port_forward() {
    sleep 5
    pod="$(kubectl -n stackrox get pod -l app=webhookserver -o name)"
    echo "Got pod ${pod}"
    [[ -n "${pod}" ]]
    kubectl -n stackrox wait --for=condition=ready "${pod}" --timeout=5m
    log="${ARTIFACT_DIR:-/tmp}/webhook_server_port_forward.log"
    nohup "${BASH_SOURCE[0]}" restart_webhook_server_port_forward "${pod}" 0<&- &> "${log}" &
    sleep 1
}

restart_webhook_server_port_forward() {
    local pod="$1"

    while true
    do
        echo "INFO: $(date): Starting webhook server port-forward: ${pod} 8080"
        kubectl -n stackrox port-forward "${pod}" 8080:8080 || {
            echo "WARNING: $(date): The webhook server port-forward exited with: $?"
            echo "Will restart in 5 seconds..."
            sleep 5
        }
    done
}

export_webhook_server_certs() {
    local certs_dir="$1"

    ci_export GENERIC_WEBHOOK_SERVER_CA_CONTENTS "$(cat "${certs_dir}/ca.crt")"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        usage
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
