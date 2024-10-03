#!/bin/bash

set -eu

output_dir="${1?:missing positional argument 'output_dir'}"
vulnerability_version="${2?:missing positional argument 'vulnerability_version'}"
supported_releases="${3?:missing positional argument 'supported_releases'}"

mkdir -p "$output_dir"

# Build the URL for vulnerabilities and mapping definitions.

filename=vulnerabilities.zip
filename_version="$vulnerability_version"
bundle_prefix=v4-definitions-

# Backward compatibility with previous releases where vulnerability version is a
# release number, in 4.4 the file is the single bundle, and bundle prefix is
# different.
case "$vulnerability_version" in
    4.4.*)
        filename=vulns.json.zst
        filename_version="v1"
        bundle_prefix=scanner-v4-defs-
        ;;
    4.5.*)
        filename_version="v1"
        bundle_prefix=scanner-v4-defs-
        ;;
esac

tmpdir=$(mktemp -d)

declare -A files_to_download=(
    ["$tmpdir/$filename"]="https://storage.googleapis.com/definitions.stackrox.io/v4/vulnerability-bundles/$filename_version/$filename"
    ["$tmpdir/mapping.zip"]="https://definitions.stackrox.io/v4/redhat-repository-mappings/mapping.zip"
)

# Download the files
for f in "${!files_to_download[@]}"; do
    curl --fail --silent --show-error --max-time 60 --retry 3 --create-dirs -o "$f" "${files_to_download[$f]}"
done
unzip -j "$tmpdir/mapping.zip" "repomapping/*" -d "$tmpdir" && rm "$tmpdir/mapping.zip"

# Manifest contains:
#
# - version: The vulnerability schema version, or the Y-stream-based version tag
#   for bundles prior to 4.6.
#
# - release_versions: The list of Z-stream versions this bundle supports.
#
# - created: The creation timestamp.
version=$(echo "$vulnerability_version" | grep -oE '^[0-9]+\.[0-9]+' || echo "$vulnerability_version")
cat >"$tmpdir/manifest.json" <<EOF
{
  "version": "$version",
  "created": "$(date -u -Iseconds)",
  "release_versions": "$supported_releases"
}
EOF

# Sanity check.
for f in "$tmpdir"/*.json; do
    jq empty "$f" || echo "jq processing failed for $f"
done

# Bundle creation.
zip -j "$output_dir/$bundle_prefix$version.zip" "$tmpdir"/*
