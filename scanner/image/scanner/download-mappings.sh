#!/usr/bin/env bash
#
# CAUTION: If you change this file, please be sure to update the corresponding midstream script
#
# Downloads and performs basic JSON validation for the name-to-repository and
# repository-to-cpu mapping files to be embedded in the Scanner v4 container.

set -euo pipefail

if [[ "$#" -lt "1" ]]; then
  >&2 echo "Error: please pass target directory as a command line argument."
  exit 1
fi

output_dir="$1"
shift

mkdir -p "$output_dir"

urls=(
    "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
    "https://security.access.redhat.com/data/metrics/container-name-repos-map.json"
)

for url in "${urls[@]}"; do
    filename=$(basename "$url")
    echo "Downloading ${url} > ${output_dir}/$filename"
    curl --location --silent --fail --show-error --retry 3 --max-time 60 \
        --output "${output_dir}/$filename" "$url"
    if [[ ! (-s "${output_dir}/$filename") ]]; then
        echo "${output_dir}/$filename is empty"
        exit 1
    fi

    if command -v python &>/dev/null; then
        echo "Validating if ${output_dir}/$filename contains parseable JSON"
        python -c '
import json, sys
with open(sys.argv[1]) as f:
    json.load(f)' "${output_dir}/$filename"
        exit_status="$?"
        if [[ "$exit_status" != "0" ]]; then
            echo "${output_dir}/$filename is not valid JSON"
            exit 1
        else
            echo "Validated"
        fi
    else
        echo "WARNING: Could not find a method to validate if the file contains parseable JSON"
    fi
done

echo "Done"
