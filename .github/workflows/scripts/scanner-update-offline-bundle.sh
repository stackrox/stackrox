#!/bin/bash

set -eu

SCANNER_V4_DEFS_BUCKET="https://storage.googleapis.com/definitions.stackrox.io"
ROX_PRODUCT_VERSION="$1"
PRODUCT_VERSION="${ROX_PRODUCT_VERSION}.0"
VULNS_FILE_NAME="$2"

declare -A files_to_download=(
    ["v4/$VULNS_FILE_NAME"]="${SCANNER_V4_DEFS_BUCKET}/v4/vulnerability-bundles/${PRODUCT_VERSION}/$VULNS_FILE_NAME"
    ["v4/mapping.zip"]="https://definitions.stackrox.io/v4/redhat-repository-mappings/mapping.zip"
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
gsutil cp "$dir/scanner-v4-defs-${ROX_PRODUCT_VERSION}.zip" "gs://definitions.stackrox.io/v4/offline-bundles/"
