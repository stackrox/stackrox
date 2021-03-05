#!/bin/bash

set -e

# Get data from a central backup.

usage() {
    echo "$0 <somewhere to put it>"
}

main() {
    if [ "$#" -ne 1 ]; then
        usage
        exit 1
    fi
    if [ -z "${ROX_PASSWORD}" ]; then
        echo "ROX_PASSWORD must be set"
        exit 1
    fi

    dest="$1"

    api_hostname=localhost
    api_port=8000
    lb_ip=$(kubectl -n stackrox get svc/central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}' || true)
    if [ -n "${lb_ip}" ]; then
        api_hostname="${lb_ip}"
        api_port=443
    fi
    api_endpoint="${api_hostname}:${api_port}"

    mkdir -p "${dest}"

    roxctl -e "${api_endpoint}" -p "${ROX_PASSWORD}" central backup --output "${dest}"

    if ! [ -x "$(command -v rocksdbdump)" ]; then
        go install ./tools/rocksdbdump
    fi

    rocksdbdump -b "${dest}"/*.zip -o "${dest}"
}

main "$@"
