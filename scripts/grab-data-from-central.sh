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

    set -vx
    local api_endpoint
    if [ -n "${API_ENDPOINT}" ]; then
        api_endpoint="${API_ENDPOINT}"
    elif [ -n "${API_HOSTNAME}" ] && [ -n "${API_PORT}" ]; then
        api_endpoint="${API_HOSTNAME}:${API_PORT}"
    else
        api_hostname=localhost
        api_port=8000
        kubectl -n stackrox get svc/central-loadbalancer -o json
        lb_ip=$(kubectl -n stackrox get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname' || true)
        if [ -n "${lb_ip}" ]; then
            api_hostname="${lb_ip}"
            api_port=443
        fi
        api_endpoint="${api_hostname}:${api_port}"
    fi

    mkdir -p "${dest}"
    set | grep '^ROX' | true
    roxctl -e "${api_endpoint}" --insecure-skip-tls-verify central backup --output "${dest}" \
      || {
      # If api_endpoint fails, try localhost forwarding:
      kubectl -n stackrox port-forward "$(kubectl get pod -n stackrox --selector 'app=central' -o name)" 8000:8443 &
      forward_pid=$!
      roxctl -e "localhost:8000" --insecure-skip-tls-verify central backup --output "${dest}"
      kill -9 "${forward_pid}";
    } || echo "Failed to connect to Central endpoint:${api_endpoint}"

    set +vx

    # Pull some data not found from the database
    set +e
    call_curl "https://${api_endpoint}/v1/imageintegrations" | jq > "${dest}/imageintegrations.json"
    for objects in "policies"; do
        echo "Pulling StackRox ${objects}"
        call_curl "https://${api_endpoint}/v1/${objects}" | jq > "${dest}/${objects}.json"

        mapfile -t object_list < <(jq -r ".${objects}[].id" < "${dest}/${objects}.json")
        echo "Will pull ${#object_list[@]} ${objects} from StackRox"

        mkdir -p "${dest}/${objects}"
        for id in "${object_list[@]}"; do
            call_curl "https://${api_endpoint}/v1/${objects}/${id}" | jq > "${dest}/${objects}/${id}.json"
        done
    done
}

main "$@"
