#!/usr/bin/env bash
set -eou pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

ncommits=${1:-4}

mapfile -t shas < <(git log | grep ^commit | head -"${ncommits}" | awk '{print $2}')

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
