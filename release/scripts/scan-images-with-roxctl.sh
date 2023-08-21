#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

scan_images_with_roxctl() {
    if [[ "${STACKROX_CI_INSTANCE_CENTRAL_HOST}" == "disabled" ]]; then
        if is_GITHUB_ACTIONS; then
            echo "::warning ::Image scan with roxctl is disabled"
        else
            info "Image scan with roxctl is disabled"
        fi
        return 0
    fi

    info "Will scan anticipated release images with roxctl"

    local images=()

    # determine all image tags
    local release_tag
    release_tag=$(make tag)
    local collector_tag
    collector_tag=$(make collector-tag)
    local scanner_tag
    scanner_tag=$(make scanner-tag)

    # check main images
    images+=("main:$release_tag")
    images+=("central-db:$release_tag")

    # check collector images
    images+=("collector:${collector_tag}-slim")
    images+=("collector:${collector_tag}")

    # check scanner images
    images+=("scanner:$scanner_tag")
    images+=("scanner-slim:$scanner_tag")

    # check scanner-db images
    images+=("scanner-db:$scanner_tag")
    images+=("scanner-db-slim:$scanner_tag")

    export ROX_API_TOKEN="${STACKROX_CI_INSTANCE_API_KEY}"
    local errors=()
    for image in "${images[@]}"; do
        roxctl image check --insecure-skip-tls-verify --endpoint "https://${STACKROX_CI_INSTANCE_CENTRAL_HOST}:443" --image "quay.io/rhacs-eng/${image}" || {
            errors+=("$image")
        }
    done

    if [[ "${#errors[@]}" != "0" ]]; then
        info "ERROR: Image check errors were found with:"
        for error in "${errors[@]}"; do
            echo -e "\t$error"
        done
        return 1
    fi

    return 0
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    scan_images_with_roxctl "$@"
fi
