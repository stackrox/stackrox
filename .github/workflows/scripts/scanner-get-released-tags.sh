#!/usr/bin/env bash
#
# Print a JSON containing all Scanner V4 vulnerability bundle streams,
# as directed by `scanner/updater/version/VULNERABILITY_BUNDLE_VERSION`.

set -euo pipefail

# Initialize the version map and json_array
declare -A version_map
json_array=()

# Populate the version_map with version-to-tag mappings
while read -r ver tag _; do
    version_map[$ver]="$tag"
done < <(./.github/workflows/scripts/scanner-output-release-versions.sh | jq -r '.versions[] | "\(.version) \(.tag)"')

# Read the versions and their corresponding tags from `scanner/updater/version/VULNERABILITY_BUNDLE_VERSION`.
while read -r line; do
    # Skip lines that are comments or empty
    echo "$line" | grep -qE '^\s*(#.*|$)' && continue

    read -r version ref <<< "$line"

    case $ref in
        heads/*)
            resolved_tag="${ref#heads/}"
            ;;
        tags/*)
            resolved_tag="${ref#tags/}"
            ;;
        *)
            # Assume anything else is a StackRox release
            resolved_tag=${version_map[$ref]:-}
            ;;
    esac

    # Check if resolved_tag is set, otherwise throw an error
    if [ -z "${resolved_tag}" ]; then
        echo >&2 "error: invalid reference in VULNERABILITY_BUNDLE_VERSIONS: $ref"
        exit 1
    fi

    # Add the JSON object to the array
    json_array+=("{\"version\":\"$version\",\"ref\":\"$resolved_tag\"}")
done < scanner/updater/version/VULNERABILITY_BUNDLE_VERSION

# Convert the array to a single-line JSON array
json_output=$(printf "%s," "${json_array[@]}" | sed 's/,$//')
json_output="[$json_output]"

# Print the final JSON output
echo "$json_output" | jq
