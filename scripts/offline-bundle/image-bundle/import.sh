#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

print_docs_usage() {
  echo
  echo "To view complete product documentation go to https://help.stackrox.com"
}

main() {
    cd "$(dirname "${BASH_SOURCE[0]}")"

    echo "This script can optionally push imported images into a private registry."
    echo "Most users add a path segment like \"/stackrox\"."
    echo "For example, you might input: my-registry.example.com:5000/stackrox"
    echo "To skip pushing, simply do not enter a prefix."
    echo -n "Enter your private registry prefix: "
    read registry_prefix
    echo

    echo "Loading main image..."
    main_tag="$(docker load -i main.img | tag)"
    main_image_local="stackrox.io/main:${main_tag}"
    main_image_remote="${registry_prefix}/main:${main_tag}"

    echo "Loading scanner images..."
    scanner_tag="$(docker load -i scanner.img | tag)"
    scanner_image_local="stackrox.io/scanner:${scanner_tag}"
    scanner_image_remote="${registry_prefix}/scanner:${scanner_tag}"

    scanner_db_tag="$(docker load -i scanner-db.img | tag)"
    scanner_db_image_local="stackrox.io/scanner-db:${scanner_db_tag}"
    scanner_db_image_remote="${registry_prefix}/scanner-db:${scanner_db_tag}"

    if [[ -z "$registry_prefix" ]]; then
        echo "No registry prefix given, all done!"
        print_docs_usage
        return
    fi

    echo "Pushing image: ${main_image_local} as ${main_image_remote}"
    docker tag "${main_image_local}" "${main_image_remote}"
    docker push "${main_image_remote}" | cat

    echo "Pushing image: ${scanner_image_local} as ${scanner_image_remote}"
    docker tag "${scanner_image_local}" "${scanner_image_remote}"
    docker push "${scanner_image_remote}" | cat

    echo "Pushing image: ${scanner_db_image_local} as ${scanner_db_image_remote}"
    docker tag "${scanner_db_image_local}" "${scanner_db_image_remote}"
    docker push "${scanner_db_image_remote}" | cat

    echo "All done!"
    print_docs_usage
}

tag() {
    sed 's/.*:\(.*$\)/\1/'
}

main
