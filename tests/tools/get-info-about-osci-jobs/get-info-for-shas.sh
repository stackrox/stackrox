#!/usr/bin/env bash
set -eou pipefail

ncommit=NA
sha=NA

process_arg() {
    arg=$1

    key="$(echo "$arg" | cut -d "=" -f 1)"
    value="$(echo "$arg" | cut -d "=" -f 2)"

    echo "key= $key"

    if [[ "$key" == "ncommit" ]]; then
        ncommit="$value"
    elif [[ "$key" == "sha" ]]; then
	sha="$value"
    fi
}

process_args() {
    echo "In process_args"
    for arg in "$@"; do
	echo "arg=$arg"
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
        "$DIR/get-info-about-osci-jobs.sh" "$temp_dir" >> $output
	tail -n +2 "$output" >> "$temp_file"
	header="$(head -1 $output)"
    else
        echo "WARNING: Unable to get artifacts for $sha"
    fi

    rm -rf "$temp_dir" || true
done

echo "$header"
sort -u "$temp_file"
