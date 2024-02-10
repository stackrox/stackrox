#!/bin/bash

set -euo pipefail

if [[ "$#" -lt "1" ]]; then
  >&2 echo "Error: please pass target directory as a command line argument."
  exit 1
fi

output_dir="$1"
shift

mkdir -p "$output_dir"

urls=(
    "https://access.redhat.com/security/data/metrics/repository-to-cpe.json"
    "https://access.redhat.com/security/data/metrics/container-name-repos-map.json"
)

for url in "${urls[@]}"; do
    filename=$(basename "$url")
    echo "Downloading ${url} > ${output_dir}/$filename"
    curl --retry 3 -sS --fail -o "${output_dir}/$filename" "$url"
done
