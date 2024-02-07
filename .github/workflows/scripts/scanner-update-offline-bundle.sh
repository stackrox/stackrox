#!/bin/bash

set -eu

SCANNER_V4_DEFS_BUCKET="https://storage.googleapis.com/scanner-v4-test"
ROX_PRODUCT_VERSION="$1"
PRODUCT_VERSION="${ROX_PRODUCT_VERSION}.0"
declare -A files_to_download=(
    ["v4/vulns.json.zst"]="${SCANNER_V4_DEFS_BUCKET}/vulnerability-bundles/${PRODUCT_VERSION}/vulns.json.zst"
    ["v4/mapping.zip"]="https://storage.googleapis.com/definitions.stackrox.io/redhat-repository-mappings/mapping.zip"
)

# Download the files
for f in "${!files_to_download[@]}"; do
    curl --fail --silent --show-error --max-time 60 --retry 3 --create-dirs -o "$f" "${files_to_download[$f]}"
done

unzip -j "v4/mapping.zip" "repomapping/*" -d v4 && rm v4/mapping.zip

for f in v4/*.json; do
  jq empty "$f" || echo "jq processing failed for $f"
done

dir=out
mkdir -p $dir
jq -n \
    --arg version "$ROX_PRODUCT_VERSION" \
    --arg date "$(date -u -Iseconds)" \
    '{"version": $version, "created": $date}' > v4/manifest.json
zip -j "$dir/scanner-v4-defs-${ROX_PRODUCT_VERSION}.zip" v4/*
gsutil cp "$dir/scanner-v4-defs-${ROX_PRODUCT_VERSION}.zip" "gs://scanner-v4-test/offline-bundles/"
