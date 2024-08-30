#!/usr/bin/env bash
#
# Print a JSON containing all scanner vulnerability bundle streams, as directed
# by the `VULNERABILITY_BUNDLE_VERSION` file.

set -euo pipefail

# Save all tags from the repo to a temporary file.
tags=$(mktemp)
git tag > "$tags"

# Function to resolve the tag associated with a vulnerability updater version.
# This logic is duplicated from scanner-output-release-versions.sh. An optimization is needed to eliminate this duplication.
# TODO: ROX-25999
resolve_tag() {
    local ver=${1:?"missing required argument: version"}
    local tag=${2:?"missing required argument: tag"}

    # Check if the tag exists in the repository
    if grep -qx "$tag" "$tags"; then
        echo "$tag"
        return
    fi

    echo >&2 "WARNING: Tag '$tag' is not a valid tag in the repository"

    # If the tag is not in the form X.Y.Z, return it as is
    if ! echo "$tag" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        echo "$tag"
        return
    fi

    # Try to find a related tag (release candidate or patch branch)
    local re
    re="${tag//./\\.}"
    re="^$re-rc\.[0-9]+|${re%.*}.x$"
    local resolved_tag
    resolved_tag=$(grep -E "^$re$" "$tags" | sort -rV | head -n 1)
    if [ -z "$resolved_tag" ] ; then
        echo >&2 "WARNING: Could not find a matching tag for version '$ver' with tag '$tag'"
        echo "$tag"
        return
    fi
    echo "$resolved_tag"
}

# Prepare an array to store the JSON objects
json_array=()

# Read the versions and their corresponding tags from the VULNERABILITY_BUNDLE_VERSION file.
while IFS=, read -r version ref; do
    # Skip lines that are comments or empty
    echo "$version" | grep -qE '^\s*(#.*|$)' && continue
    ref=$(echo "$ref" | xargs)

    # Check prefix
    if [[ $ref == heads/* ]]; then
        # Extract branch name from "heads/<branch-name>"
        resolved_tag="${ref#heads/}"
    elif [[ $ref == tags/* ]]; then
        # Extract tag name from "tags/<tag-name>"
        resolved_tag="${ref#tags/}"
    else
        # Resolve tag as before for tags with no prefix
        resolved_tag=$(resolve_tag "$version" "$ref")
    fi

    # Add the JSON object to the array
    json_array+=("{\"version\":\"$version\",\"tag\":\"$resolved_tag\"}")
done < scanner/updater/version/VULNERABILITY_BUNDLE_VERSION


# Convert the array to a single-line JSON array
json_output=$(printf "%s," "${json_array[@]}" | sed 's/,$//')
json_output="[$json_output]"

# Print the final JSON output
echo "$json_output"

# Clean up temporary file
rm "$tags"
