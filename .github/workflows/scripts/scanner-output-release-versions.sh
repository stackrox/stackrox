#!/usr/bin/env bash
#
# Print a JSON object mapping each target release version to its repository tag
# for Scanner, as directed by the `scanner/updater/version/RELEASE_VERSION` file
# and the `SCANNER_RELEASE_ALLOW_RC` environment variable.

set -euo pipefail

# Save all tags from the repo to a temporary file.

tags=$(mktemp)
git tag > "$tags"

# tag determines the tag associated with a vulnerability updater version and
# prints it.
tag() {
    local ver=${1:?"missing required argument: version"}

    # Happy case: The release version is simply a tag in the repository.
    if grep -qx "$ver" "$tags"; then
        echo "$ver"
        return 0
    fi

    echo >&2 "WARNING: Version '$ver' is not a tag in the repository"

    # Sanity check: Fail open if this doesn't look like a release version.
    if ! echo "$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        echo >&2 "ERROR: Version '$ver' is not in X.Y.Z format"
        return 1
    fi

    # Fail open if release candidate matching is disabled.
    local use_rc="${SCANNER_RELEASE_ALLOW_RC:-false}"
    if [[ "$use_rc" != "true" ]]; then
        echo >&2 "INFO: Release candidate matching is disabled (SCANNER_RELEASE_ALLOW_RC='$use_rc')"
        return 1
    fi

    # Find the latest release candidate for that release version.
    local re tag
    re="^${ver//./\\.}-rc\.[0-9]+$"
    tag=$(grep -E "$re" "$tags" | sort -rV | head -n 1)
    if [ -z "$tag" ]; then
        echo >&2 "ERROR: Could not find an RC tag for version '$ver', failing open..."
        return 1
    fi

    echo "$tag"
    return 0
}

# Go over all release versions and generate the JSON.

echo '{"versions": ['

while IFS= read -r version; do
    echo "$version" | grep -qE '^\s*(#.*|$)' && continue
    tag_value=$(tag "$version") || continue
    cat <<EOF
  {"tag": "$tag_value", "version": "$version"},
EOF
done <scanner/updater/version/RELEASE_VERSION | sed '$ s/,$//'

echo ']}'
