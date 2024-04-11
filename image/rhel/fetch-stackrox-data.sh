#!/usr/bin/env bash

# Fetches data used by the stackrox:main image

set -euxo pipefail

TARGET_DIR="${1:-/stackrox-data}"

fetch_stackrox_data() {
    local target_dir="${1}"

    mkdir -p /tmp/external-networks
    local latest_prefix
    latest_prefix="$(curl --fail https://definitions.stackrox.io/external-networks/latest_prefix)"
    curl --fail --output /tmp/external-networks/checksum "https://definitions.stackrox.io/${latest_prefix}/checksum"
    test -s /tmp/external-networks/checksum

    curl --fail --output /tmp/external-networks/networks "https://definitions.stackrox.io/${latest_prefix}/networks"
    test -s /tmp/external-networks/networks

    sha256sum -c <( echo "$(cat /tmp/external-networks/checksum) /tmp/external-networks/networks" )

    mkdir -p "${target_dir}/external-networks"
    zip -jr --test "${target_dir}/external-networks/external-networks.zip" /tmp/external-networks
    rm -rf /tmp/external-networks
}

fetch_stackrox_data "${TARGET_DIR}"
