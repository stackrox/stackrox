#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

FAIL_FLAG="$(mktemp)"
trap 'rm -f $FAIL_FLAG' EXIT

check_example_rpmdb_files_are_ignored() {
    # At the time of this writing, Konflux uses syft to generate SBOMs for built containers.
    # If we happen to have test rpmdb databases in the repo, syft will union their contents with RPMs that it finds
    # installed in the container resulting in a misleading SBOM.
    # This check is to make sure the exclusion list in Syft config enumerates all such rpmdbs.
    # Ref https://github.com/anchore/syft/wiki/configuration
    # TODO: the check can be removed after KONFLUX-3515 is implemented.

    local -r syft_config=".syft.yaml"
    local -r exclude_attribute=".exclude"

    local actual_excludes
    actual_excludes="$(yq eval "${exclude_attribute}" "${syft_config}")"

    local expected_excludes
    expected_excludes="$(git ls-files -- '**/rpmdb.sqlite' | sort | uniq | sed 's/^/- .\//')"

    echo
    echo "➤ ${syft_config} // checking ${exclude_attribute}: all rpmdb files in the repo shall be mentioned."
    if ! compare "${expected_excludes}" "${actual_excludes}"; then
        echo >&2 "How to resolve:
1. Open ${syft_config} and replace ${exclude_attribute} contents with the following.
${expected_excludes}"
        record_failure "${FUNCNAME}"
    fi
}

compare() {
    local -r expected="$1"
    local -r actual="$2"

    if ! diff --brief <(echo "${expected}") <(echo "${actual}") > /dev/null; then
        echo >&2 "✗ ERROR: the expected contents (left) don't match the actual ones (right):"
        diff >&2 --side-by-side <(echo "${expected}") <(echo "${actual}") || true
        return 1
    else
        echo "✓ No diff detected."
    fi
}

record_failure() {
    local -r func="$1"
    echo "${func}" >> "${FAIL_FLAG}"
}

echo "Checking our Konflux pipelines and builds setup."
check_example_rpmdb_files_are_ignored

if [[ -s "$FAIL_FLAG" ]]; then
    echo >&2
    echo >&2 "✗ Some Konflux checks failed:"
    cat >&2 "$FAIL_FLAG"
    exit 1
else
    echo
    echo "✓ All checks passed."
fi
