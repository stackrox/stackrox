#!/usr/bin/env bash
#
# Print a JSON containing all Scanner V4 vulnerability bundle streams, as
# directed by `scanner/updater/version/VULNERABILITY_BUNDLE_VERSION`, or
# optionally, specified environment variables:
#
# 1. If no envs are not specified, use
# `scanner/updater/version/VULNERABILITY_BUNDLE_VERSION`
#
# 2. If SCANNER_BUNDLE_REFERENCE is specified, fetch the bundle stream version
#    from the VULNERABILITY_VERSION file for that reference.
#
# 3. If both SCANNER_BUNDLE_REFERENCE and SCANNER_BUNDLE_STREAM, use both.
#
# The reference in any of the above needs to be a valid git reference, or a
# special reference type called "release version" on the form of X.Y.Z string,
# which will be automatically resolved to the git reference for that release.
#
# When resolving "release version" references, you can optionally use
# SCANNER_RELEASE_ALLOW_RC to resolve them to release candidates when the actual
# release tag does not exist.

set -euo pipefail

# Where to fetch bundle stream version specifications.
versions_file="scanner/updater/version/VULNERABILITY_BUNDLE_VERSION"

# If set, skip versions_file and use the values in the environment.
custom_ref="${SCANNER_BUNDLE_REFERENCE:-}"
custom_version="${SCANNER_BUNDLE_STREAM:-}"

# Accept RC candidates for X.Y.Z "release version" references, appending `-rc`
# to vulnerability bundle stream version.
use_rc="${SCANNER_RELEASE_ALLOW_RC:-false}"

# We use a special version '-' to signal we should use the value fetched from
# the tag.  This is supported even for the versions file.
empty_version="-"

# fetch_version() fetches the vulnerability stream version used by a given tag.
fetch_version() {
    local tag="${1:?missing positional argument 'tag'}"

    # TODO This logic is duplicated in the offline bundle matrix, we should
    #      factor this out.

    # Get the version from the VULNERABILITY_VERSION used by that reference.
    url=$(printf 'https://raw.githubusercontent.com/stackrox/stackrox/%s/scanner/VULNERABILITY_VERSION' "$tag")
    echo >&2 "INFO: fetching vulnerability version for tag '$tag' from '$url'"

    local ver

    ver=$(curl --fail --silent --show-error --max-time 30 --retry 3 "$url")
    if [ -z "$ver" ]; then
        echo >&2 "ERROR: failed to read vulnerability stream version: URL returned empty"
        return 1
    fi

    echo >&2 "INFO: setting SCANNER_BUNDLE_STREAM to '$ver'"
    echo "$ver"
}

# Validate input.
if [ -z "$custom_ref" ] && [ -n "$custom_version" ]; then
    echo >&2 "ERROR: SCANNER_BUNDLE_STREAM is set, but SCANNER_BUNDLE_REFERENCE is empty"
    exit 1
fi
if ! [ -f "$versions_file" ]; then
    echo >&2 "ERROR: versions file '$versions_file' does not exist."
    exit 1
fi

# Initialize the version map and json_array.
declare -A version_map
json_array=()

# Populate the version_map with version-to-tag mappings.
while read -r ver tag _; do
    version_map[$ver]="$tag"
done < <(./.github/workflows/scripts/scanner-output-release-versions.sh | jq -r '.versions[] | "\(.version) \(.tag)"')

# Read the versions and their corresponding tags from `$versions_file` or env
# (see redirect below).
while read -r line; do
    # Skip lines that are comments or empty
    echo "$line" | grep -qE '^\s*(#.*|$)' && continue

    read -r version ref <<< "$line"

    # We only append `-rc` if reference is a "release version" reference, and
    # use RC is true, setting to false by default.
    append_rc="false"

    case $ref in
        heads/*)
            resolved_tag="${ref#heads/}"
            ;;
        tags/*)
            resolved_tag="${ref#tags/}"
            ;;
        *)
            # Assume anything else is a release version reference.
            resolved_tag=${version_map[$ref]:-}
            [[ "$use_rc" == "true" ]] && append_rc="true"
            ;;
    esac

    # Check if resolved_tag is set, otherwise throw an error
    if [ -z "${resolved_tag}" ]; then
        echo >&2 "ERROR: invalid reference in VULNERABILITY_BUNDLE_VERSIONS: $ref"
        exit 1
    fi

    case "$version" in
        dev|v1)
            # These vulnerability bundle versions are special, and we cannot
            # validate them by fetching the vulnerability version.
            ;;
        *)
            actual_version=$(fetch_version "$resolved_tag")
            # If empty, we adopt what fetch is giving us.
            [[ "$version" == "$empty_version" ]] && version="$actual_version"
            if [[ "$version" != "$actual_version" ]]; then
                echo >&2 "ERROR: the specified version '$version' does not match the tag's: $actual_version"
                exit 1
            fi
            ;;
    esac

    if [[ "$append_rc" == "true" ]]; then
        version="$version-rc"
    fi

    # Add the JSON object to the array
    json_array+=("{\"version\":\"$version\",\"ref\":\"$resolved_tag\"}")
done < <(
    if [ -n "$custom_ref" ]; then
        [ -z "$custom_version" ] && custom_version="$empty_version"
        echo "$custom_version" "$custom_ref"
    else
        cat $versions_file
    fi
)

# Convert the array to a single-line JSON array
json_output=$(printf "%s," "${json_array[@]}" | sed 's/,$//')
json_output="[$json_output]"

# Print the final JSON output
echo "$json_output" | jq
