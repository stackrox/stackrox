#!/usr/bin/env bash

# Fetches data used by the stackrox:main image

set -euxo pipefail

fetch_stackrox_data() {
    mkdir -p /tmp/external-networks
    local latest_prefix
    latest_prefix="$(curl --fail https://definitions.stackrox.io/external-networks/latest_prefix)"
    curl --fail --output /tmp/external-networks/checksum "https://definitions.stackrox.io/${latest_prefix}/checksum"
    test -s /tmp/external-networks/checksum

    curl --fail --output /tmp/external-networks/networks "https://definitions.stackrox.io/${latest_prefix}/networks"
    test -s /tmp/external-networks/networks

    sha256sum -c <( echo "$(cat /tmp/external-networks/checksum) /tmp/external-networks/networks" )

    mkdir /stackrox-data/external-networks
    zip -jr /stackrox-data/external-networks/external-networks.zip /tmp/external-networks
    rm -rf /tmp/external-networks
}

fetch_stackrox_data
