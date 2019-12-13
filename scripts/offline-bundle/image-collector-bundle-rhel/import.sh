#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

main() {
    cd "$(dirname "${BASH_SOURCE[0]}")"

    echo "This script can optionally push imported images into a private registry."
    echo "Most users add a path segment like \"/stackrox\"."
    echo "For example, you might input: my-registry.example.com:5000/stackrox"
    echo "To skip pushing, simply do not enter a prefix."
    echo -n "Enter your private registry prefix: "
    read registry_prefix
    echo

    echo "Loading collector image..."
    collector_tag="$(docker load -i collector.img | tag)"
    collector_image_local="stackrox.io/collector-rhel:${collector_tag}"
    collector_image_remote="${registry_prefix}/collector-rhel:${collector_tag}"

    if [[ -z "$registry_prefix" ]]; then
        echo "No registry prefix given, all done!"
        return
    fi

    echo "Pushing image: ${collector_image_local} as ${collector_image_remote}"
    docker tag "${collector_image_local}" "${collector_image_remote}"
    docker push "${collector_image_remote}" | cat

    echo "All done!"
}

tag() {
    sed 's/.*:\(.*$\)/\1/'
}

main
