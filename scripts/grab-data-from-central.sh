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
    if [ -z "${ROX_USERNAME}" ] || [ -z "${ROX_PASSWORD}" ]; then
        echo "ROX_USERNAME and ROX_PASSWORD must be set"
        exit 1
    fi

    dest="$1"

    local api_endpoint
    if [ -n "${API_ENDPOINT}" ]; then
        api_endpoint="${API_ENDPOINT}"
    elif [ -n "${API_HOSTNAME}" ] && [ -n "${API_PORT}" ]; then
        api_endpoint="${API_HOSTNAME}:${API_PORT}"
    else
        api_hostname=localhost
        api_port=8000
        lb_ip=$(kubectl -n stackrox get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname' || true)
        if [ -n "${lb_ip}" ]; then
            api_hostname="${lb_ip}"
            api_port=443
        fi
        api_endpoint="${api_hostname}:${api_port}"
    fi

    mkdir -p "${dest}"

    roxctl -e "${api_endpoint}" -p "${ROX_PASSWORD}" --insecure-skip-tls-verify central backup --output "${dest}"

    # With Postgres we no longer take RocksDB dumps
    if [ -z "${ROX_POSTGRES_DATASTORE}" ] || [ "${ROX_POSTGRES_DATASTORE}" == "false" ]; then
      if ! [ -x "$(command -v rocksdbdump)" ]; then
          go install ./tools/rocksdbdump
      fi

      rocksdbdump -b "${dest}"/*.zip -o "${dest}"
    fi

    # Pull some data not found from the database

    set +e
    curl -s --insecure -u "${ROX_USERNAME}:${ROX_PASSWORD}" "https://${api_endpoint}/v1/imageintegrations" | jq > "${dest}/imageintegrations.json"
    for objects in "policies"; do
        echo "Pulling StackRox ${objects}"
        curl -s --insecure -u "${ROX_USERNAME}:${ROX_PASSWORD}" "https://${api_endpoint}/v1/${objects}" | jq > "${dest}/${objects}.json"

        mapfile -t object_list < <(jq -r ".${objects}[].id" < "${dest}/${objects}.json")
        echo "Will pull ${#object_list[@]} ${objects} from StackRox"

        mkdir -p "${dest}/${objects}"
        for id in "${object_list[@]}"; do
            curl -s --insecure -u "${ROX_USERNAME}:${ROX_PASSWORD}" "https://${api_endpoint}/v1/${objects}/${id}" | jq > "${dest}/${objects}/${id}.json"
        done
    done
}

main "$@"
