#!/usr/bin/env bash

# Fetches data used by the stackrox:main image

set -euo pipefail

fetch_stackrox_data() {
    mkdir -p /tmp/external-networks
    local latest_prefix
    latest_prefix="$(curl https://definitions.stackrox.io/external-networks/latest_prefix)"
    curl "https://definitions.stackrox.io/${latest_prefix}/checksum" > /tmp/external-networks/checksum
    test -s /tmp/external-networks/checksum

    curl "https://definitions.stackrox.io/${latest_prefix}/networks" > /tmp/external-networks/networks
    test -s /tmp/external-networks/networks

    echo -n " /tmp/external-networks/networks" >> /tmp/external-networks/checksum
    sha256sum -c /tmp/external-networks/checksum

    mkdir /stackrox-data/external-networks
    zip -jr /stackrox-data/external-networks/external-networks.zip /tmp/external-networks
    rm -rf /tmp/external-networks
}

fetch_stackrox_data
