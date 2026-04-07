#!/bin/bash

set -eu

output_dir="${1?:missing positional argument 'output_dir'}"
vulnerability_version="${2?:missing positional argument 'vulnerability_version'}"
supported_releases="${3?:missing positional argument 'supported_releases'}"
if ! command -v parallel &>/dev/null; then
    echo missing needed binary: 'parallel(1)' >&2
    exit 1
fi

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

declare -a files_to_download=(
    "https://storage.googleapis.com/definitions.stackrox.io/v4/vulnerability-bundles/$filename_version/$filename"
    "https://definitions.stackrox.io/v4/redhat-repository-mappings/mapping.zip"
)

# Download the files. The vulnerability bundle is ~317MB and may take longer than
# 60 seconds on slower network conditions between GitHub runners and GCS.
curl -K - <<-.
    parallel
    fail
    silent
    show-error
    max-time=300
    retry=3
    output-dir="${tmpdir}"
    remote-name
    $(printf 'url="%s"\n' "${files_to_download[@]}")
.
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
parallel --tag --null jq empty ::: "$tmpdir"/*.json

# Bundle creation.
zip -j "$output_dir/$bundle_prefix$version.zip" "$tmpdir"/*
