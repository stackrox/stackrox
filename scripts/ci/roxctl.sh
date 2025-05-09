#!/bin/bash

set -e

# A wrapper for roxctl command execution in CI.

usage() {
    echo "$0 <other roxctl args>"
}

main() {
    if [ "$#" -eq 0 ]; then
        usage
        exit 1
    fi
    if [ -z "${ROX_ADMIN_PASSWORD}" ]; then
        echo "ROX_ADMIN_PASSWORD must be set"
        exit 1
    fi

    api_hostname=localhost
    api_port=8000
    lb_ip=$(kubectl -n stackrox get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname' || true)
    if [ -n "${lb_ip}" ]; then
        api_hostname="${lb_ip}"
        api_port=443
    fi
    api_endpoint="${api_hostname}:${api_port}"

    set -x
    echo "api_endpoint:$api_endpoint"
    echo "API_ENDPOINT:$API_ENDPOINT"
    echo "ROX_SERVER_NAME:$ROX_SERVER_NAME"
    roxctl -e "${api_endpoint}" --insecure-skip-tls-verify "$@" \
      || { echo 'retry without ROX_SERVER_NAME'; ROX_SERVER_NAME='' roxctl -e "${api_endpoint}" --insecure-skip-tls-verify "$@"; }
    set +x
}

main "$@"
