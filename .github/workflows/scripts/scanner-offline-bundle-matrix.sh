#!/usr/bin/env bash
#
# Print the mapping of StackRox releases to scanner vulnerability bundle schema
# versions. One entry per line, separated by whitespace.

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# GITHUB_URL_PATTERN points to vulnerability version file for a given branch/tag.
GITHUB_URL_PATTERN="https://raw.githubusercontent.com/stackrox/stackrox/%s/scanner/VULNERABILITY_VERSION"

RELEASE_VERSION_FILE="scanner/updater/version/RELEASE_VERSION"

[ -f $RELEASE_VERSION_FILE ] || {
    echo "error: release version file does not exist: $RELEASE_VERSION_FILE"
    exit 1
}

# version_object() prints a JSON object for a bundle version and its supported releases.
version_object() {
    local vuln_version=${1?:"missing positional argument 'vuln_version'"}
    local releases=${2?:"missing positional argument 'releases'"}
    cat <<EOF
  {
    "vulnerability_version": "$vuln_version",
    "supported_releases": "$releases"
  }
EOF

}

main() {
    # Iterate over all tags that look like StackRox release tags, and output a JSON
    # array.
    local vuln_version
    local vuln_version_url

    declare -A releases

    while read -r tag version _; do
        # Sanity check.
        if echo "$version" | grep -q '^[0-4]\.[0-3]'; then
            echo >&2 "info: skipping pre-V4 tag: $version"
            continue
        fi
        case "$version" in
            4.4.*|4.5.*)
                # This is hard-coded to "v1", the initial vulnerability schema
                # version.
                vuln_version="v1"
                ;;
            *)
                # Check the vuln version from the repository.
                # shellcheck disable=SC2059
                vuln_version_url=$(printf "$GITHUB_URL_PATTERN" "$tag")
                echo >&2 "info: get vulnerability version: $vuln_version_url"
                vuln_version=$(curl \
                                   --fail \
                                   --silent \
                                   --show-error \
                                   --max-time 30 \
                                   --retry 3 \
                                   "$vuln_version_url")
                if [[ -z "$vuln_version" ]]; then
                    echo >&2 "error: failed to read vulnerability version: URL returned empty"
                    exit 1
                fi
                ;;
        esac
        # Words separated by white spaces.
        releases[$vuln_version]+="$version "
    done < <("$DIR"/scanner-output-release-versions.sh | jq -r '.versions[] | "\(.tag) \(.version)"')
    echo '{"versions": ['
    for v in "${!releases[@]}"; do
        version_object "$v" "${releases[$v]% *}"
        echo ,
    done

    # Manual entries for backward compatibility with previous releases, where
    # offline bundle version is tied to the release, not the vulnerability
    # schema.
    version_object "4.4.0" "$(grep ^4.4 scanner/updater/version/RELEASE_VERSION | paste -sd ' ')"
    echo ,
    version_object "4.5.0" "$(grep ^4.5 scanner/updater/version/RELEASE_VERSION | paste -sd ' ')"
    echo ']}'
}

main
