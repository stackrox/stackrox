#!/usr/bin/env bash
set -eou pipefail

# Runs the script get-info-about-osci-jobs.sh to get infromation about which kernel versions and
# collection methods were used for OSCI jobs, from artifacts for multiple CI runs. The artifacts
# are obtained by downloading them from gcp buckets. The buckets are specified with commit shas.
#
# The output is a set of csv files and stdout with only unique lines from the set of csv files
#
# There are two options for the command line ncommit and sha
#
# ncommit gets the SHAs for the past ncommit and gets the artifacts for them.
# sha adds a specific SHA to the list of SHAs to process.
# If neither option is used ncommit is set to 6.
#
# Example usage:
# ./get-info-for-shas.sh ncommit=4 sha=3c7bc3b7e08d11eeef2122c7b3ea801db4e07599

ncommit=NA
sha=NA

process_arg() {
    arg=$1

    key="$(echo "$arg" | cut -d "=" -f 1)"
    value="$(echo "$arg" | cut -d "=" -f 2)"

    if [[ "$key" == "ncommit" ]]; then
        ncommit="$value"
    elif [[ "$key" == "sha" ]]; then
        sha="$value"
    fi
}

process_args() {
    for arg in "$@"; do
        process_arg "$arg"
    done
}

DIR="$(cd "$(dirname "$0")" && pwd)"

process_args "$@"

if [[ "$ncommit" == "NA" && "$sha" == "NA" ]]; then
    ncommit=6
fi

shas=()

if [[ "$ncommit" != "NA" ]]; then
    mapfile -t shas < <(git log | grep ^commit | head -"${ncommit}" | awk '{print $2}')
fi

if [[ "$sha" != "NA" ]]; then
    shas+=("$sha")
fi

for sha in "${shas[@]}"; do
    output="OSCI_Collector_Info_${sha}.csv"
    temp_dir="$(mktemp -d)"
    temp_file="$(mktemp)"

    error_code=0
    gsutil -m cp -r "gs://roxci-artifacts/stackrox/$sha" "$temp_dir" || error_code=$?
    if (( error_code == 0 )); then
        "$DIR/get-info-about-osci-jobs.sh" "$temp_dir" >> "$output"
        tail -n +2 "$output" >> "$temp_file"
        header="$(head -1 "$output")"
    else
        echo "WARNING: Unable to get artifacts for $sha"
    fi

    rm -rf "$temp_dir" || true
done

echo "$header"
sort -u "$temp_file"
