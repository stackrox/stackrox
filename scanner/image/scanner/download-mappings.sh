#!/bin/bash

set -euo pipefail

output_dir="/mappings"
mkdir $output_dir

urls=(
    "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
    "https://security.access.redhat.com/data/metrics/container-name-repos-map.json"
)

for url in "${urls[@]}"; do
    filename=$(basename "$url")
    echo "Downloading ${url} > ${output_dir}/$filename"
    curl --retry 3 -sS --fail -o "${output_dir}/$filename" "$url"
done

echo "Done"
