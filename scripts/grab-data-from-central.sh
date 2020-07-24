#!/bin/bash

# Get data from various API endpoints.

usage() {
    echo "$0 <somewhere to put it>"
}

main() {
    if [ "$#" -ne 1 ]; then
        usage
        exit 1
    fi
    if [ -z "${ROX_USERNAME}" -o -z "${ROX_PASSWORD}" ]; then
        echo "ROX_USERNAME and ROX_PASSWORD must be set"
        exit 1
    fi

    set +e

    dest="$1"

    api_hostname=localhost
    api_port=8000
    lb_ip=$(kubectl -n stackrox get svc/central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}' || true)
    if [[ ! -z "${lb_ip}" ]]; then
        api_hostname="${lb_ip}"
        api_port=443
    fi
    api_endpoint="${api_hostname}:${api_port}"

    mkdir -p ${dest}

    curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://${api_endpoint}/v1/images | jq > ${dest}/images.json
    curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://${api_endpoint}/v1/imageintegrations | jq > ${dest}/imageintegrations.json
    curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://${api_endpoint}/v1/deployments | jq > ${dest}/deployments.json

    for objects in "images" "deployments"; do
        jq_tweezer=".${objects}[].id"
        object_list=$(cat ${dest}/${objects}.json | jq "${jq_tweezer}")

        mkdir -p ${dest}/${objects}
        for id in ${object_list}; do
            id=$(echo ${id} | sed s/\"//g)
            curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://${api_endpoint}/v1/${objects}/${id} | jq > ${dest}/${objects}/${id}.json
        done
    done
}

main "$@"
