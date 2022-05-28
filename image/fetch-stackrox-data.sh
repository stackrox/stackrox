#!/usr/bin/env bash

# Fetches data used by the stackrox:main image

set -euo pipefail

wget_with_retries() {
    wget --tries=5 --retry-on-http-error=524 -O "$1" "$2"
}

quiet_wget_with_retries() {
    wget -q --tries=5 --retry-on-http-error=524 -O "$1" "$2"
}

fetch_stackrox_data() {
    mkdir -p /stackrox-data/cve/istio
    wget_with_retries /stackrox-data/cve/istio/checksum "https://definitions.stackrox.io/cve/istio/checksum"
    wget_with_retries /stackrox-data/cve/istio/cve-list.json "https://definitions.stackrox.io/cve/istio/cve-list.json"

    mkdir -p /tmp/external-networks
    local latest_prefix
    latest_prefix="$(quiet_wget_with_retries - https://definitions.stackrox.io/external-networks/latest_prefix)"
    wget_with_retries /tmp/external-networks/checksum "https://definitions.stackrox.io/${latest_prefix}/checksum"
    wget_with_retries /tmp/external-networks/networks "https://definitions.stackrox.io/${latest_prefix}/networks"
    test -s /tmp/external-networks/checksum
    test -s /tmp/external-networks/networks
    mkdir /stackrox-data/external-networks
    zip -jr /stackrox-data/external-networks/external-networks.zip /tmp/external-networks
    rm -rf /tmp/external-networks
}

fetch_stackrox_data
