#!/usr/bin/env bash
#
# Print a JSON containing all scanner vulnerability bundle streams, as directed
# by the `scanner/updater/version/RELEASE_VERSION` file.

set -euo pipefail

# Save all tags from the repo to a temporary file.

tags=$(mktemp)
git tag > "$tags"

# tag prints the tag associated with a vulnerability updater version.
tag() {
    local ver=${1:?"missing required argument: version"}
    if grep -qx "$ver" "$tags"; then
        echo "$ver"
        return
    fi
    echo >&2 "WARNING: Version '$ver' is not a tag in the repository"
    # If not X.Y.Z then don't try to a find an existing tag.
    if ! echo "$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        echo "$ver"
        return
    fi
    local re
    re="${ver//./\\.}"
    re="^$re-rc\.[0-9]+|${re%.*}.x$"
    local tag
    tag=$(grep -E "^$re$" "$tags" | sort -rV | head -n 1)
    if [ -z "$tag" ] ; then
        echo >&2 "WARNING: Could not find a matching tags for version '$ver'"
        echo "$ver"
        return
    fi
    echo "$tag"
}

# Go over all updater versions and generate the JSON.

echo '{"versions": ['

while IFS= read -r version; do
    echo "$version" | grep -qE '^\s*(#.*|$)' && continue
    cat <<EOF
  {"tag": "$(tag "$version")", "version": "$version"},
EOF
done <scanner/updater/version/RELEASE_VERSION | sed '$ s/,$//'

echo ']}'
