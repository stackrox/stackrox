#!/bin/bash

set -e

# Get data from a central backup.

usage() {
    echo "$0 <somewhere to put it>"
}

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
    echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

call_curl() {
    local url=$1
    curl -s --insecure --config <(curl_cfg user "${ROX_USERNAME}:${ROX_ADMIN_PASSWORD}") "$url"
}

main() {
    if [ "$#" -ne 1 ]; then
        usage
        exit 1
    fi
    if [ -z "${ROX_USERNAME}" ] || [ -z "${ROX_ADMIN_PASSWORD}" ]; then
        echo "ROX_USERNAME and ROX_ADMIN_PASSWORD must be set"
        exit 1
    fi

    dest="$1"

    echo "DBG 0: ${API_ENDPOINT}"
    if [ -z "${API_ENDPOINT}" ]; then
        if [ -n "${API_HOSTNAME}" ] && [ -n "${API_PORT}" ]; then
            API_ENDPOINT="${API_HOSTNAME}:${API_PORT}"
        else
            api_hostname=localhost
            api_port=8000
            lb_ip=$(kubectl -n stackrox get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname' || true)
            if [ -n "${lb_ip}" ]; then
                api_hostname="${lb_ip}"
                api_port=443
            fi
            API_ENDPOINT="${api_hostname}:${api_port}"
        fi
    fi

    if [ -z "$ROX_CA_CERT_FILE" ]; then
        # shellcheck disable=SC2034
        ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY="true"
    fi

    mkdir -p "${dest}"
    echo "DBG 0.5 API_ENDPOINT: ${API_ENDPOINT}"
    echo "DBG 1 ROX_ENDPOINT: ${ROX_ENDPOINT}"
    echo "DBG 2 ROX_CA_CERT_FILE: ${ROX_CA_CERT_FILE}"
    echo "DBG 3 ROX_SERVER_NAME: ${ROX_SERVER_NAME}"
    roxctl -e "${API_ENDPOINT}" central backup --output "${dest}"

    # Pull some data not found from the database
    set +e
    call_curl "https://${API_ENDPOINT}/v1/imageintegrations" | jq > "${dest}/imageintegrations.json"
    for objects in "policies"; do
        echo "Pulling StackRox ${objects}"
        call_curl "https://${API_ENDPOINT}/v1/${objects}" | jq > "${dest}/${objects}.json"

        mapfile -t object_list < <(jq -r ".${objects}[].id" < "${dest}/${objects}.json")
        echo "Will pull ${#object_list[@]} ${objects} from StackRox"

        mkdir -p "${dest}/${objects}"
        for id in "${object_list[@]}"; do
            call_curl "https://${API_ENDPOINT}/v1/${objects}/${id}" | jq > "${dest}/${objects}/${id}.json"
        done
    done
}

main "$@"
