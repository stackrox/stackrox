#!/usr/bin/env bash

# Fetches data used by the stackrox:main image

set -euxo pipefail

if [[ "$#" -lt "1" ]]; then
	>&2 echo "Error: please provide target directory where to store downloaded data"
	exit 6
fi

fetch_stackrox_data() {
    local target_dir="${1}"

    local download_dir
    download_dir="$(mktemp -d --tmpdir external-networks.XXXXXXXXXX)"
    local latest_prefix
    latest_prefix="$(curl --fail https://definitions.stackrox.io/external-networks/latest_prefix | sed 's/ /%20/g')"
    curl --fail --output "${download_dir}/checksum" "https://definitions.stackrox.io/${latest_prefix}/checksum"
    test -s "${download_dir}/checksum"

    curl --fail --output "${download_dir}/networks" "https://definitions.stackrox.io/${latest_prefix}/networks"
    test -s "${download_dir}/networks"

    sha256sum -c <( echo "$(cat "${download_dir}/checksum")" "${download_dir}/networks" )

    mkdir -p "${target_dir}/external-networks"
    zip -jr --test "${target_dir}/external-networks/external-networks.zip" "${download_dir}"
    rm -rf "${download_dir}"
}

fetch_stackrox_data "${1}"
