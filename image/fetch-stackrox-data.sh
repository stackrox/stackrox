#!/usr/bin/env bash

# Fetches data used by the stackrox:main image

set -euo pipefail

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

wget_with_retries() {
    retry 5 true wget -O "$1" "$2"
}

quiet_wget_with_retries() {
    retry 5 true wget -q -O "$1" "$2"
}

# retry() - retry a command up to a specific numer of times until it exits
# successfully, with exponential back off.
# (original source: https://gist.github.com/sj26/88e1c6584397bb7c13bd11108a579746)

retry() {
    if [[ "$#" -lt 3 ]]; then
        die "usage: retry <try count> <delay true|false> <command> <args...>"
    fi

    local tries=$1
    local delay=$2
    shift; shift;

    local count=0
    until "$@"; do
        exit=$?
        wait=$((2 ** count))
        count=$((count + 1))
        if [[ $count -lt $tries ]]; then
            info "Retry $count/$tries exited $exit"
            if $delay; then
                info "Retrying in $wait seconds..."
                sleep $wait
            fi
        else
            echo "Retry $count/$tries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}

fetch_stackrox_data
