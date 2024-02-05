#!/bin/bash

set -eu

SCANNER_V4_DEFS_BUCKET="https://storage.googleapis.com/scanner-v4-test"
ROX_PRODUCT_VERSION="$1"
PRODUCT_VERSION="${ROX_PRODUCT_VERSION}.0"
declare -A files_to_download=(
    ["v4/vulns.json.zst"]="${SCANNER_V4_DEFS_BUCKET}/vulnerability-bundles/${PRODUCT_VERSION}/vulns.json.zst"
    ["v4/repository-to-cpe.json"]="${SCANNER_V4_DEFS_BUCKET}/redhat-repository-mappings/repository-to-cpe.json"
    ["v4/container-name-repos-map.json"]="${SCANNER_V4_DEFS_BUCKET}/redhat-repository-mappings/container-name-repos-map.json"
    ["v2/scanner-vuln-updates.zip"]="https://storage.googleapis.com/scanner-support-public/offline/v1/scanner-vuln-updates.zip"
)

# Download the files
for f in "${!files_to_download[@]}"; do
    curl --fail --silent --show-error --max-time 60 --retry 3 --create-dirs -o "$f" "${files_to_download[$f]}"
done

for f in v4/*.json; do
  jq empty "$f" || echo "jq processing failed for $f"
done

dir=out
mkdir -p $dir

zip -j "$dir/scanner-v4-defs-${ROX_PRODUCT_VERSION}.zip" v4/*
gsutil cp "$dir/scanner-v4-defs-${ROX_PRODUCT_VERSION}.zip" "gs://scanner-v4-test/offline-bundles/"
