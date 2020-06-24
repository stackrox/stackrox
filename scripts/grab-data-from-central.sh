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

    dest="$1"

    mkdir -p ${dest}

    curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://localhost:8000/v1/images | jq > ${dest}/images.json
    curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://localhost:8000/v1/imageintegrations | jq > ${dest}/imageintegrations.json

    image_list=$(cat ${dest}/images.json | jq '.images[].id')

    mkdir -p ${dest}/images
    for image_id in ${image_list}; do
        image_id=$(echo ${image_id} | sed s/\"//g)
        curl -s --insecure -u ${ROX_USERNAME}:${ROX_PASSWORD} https://localhost:8000/v1/images/${image_id} | jq > ${dest}/images/${image_id}.json
    done
}

main "$@"
