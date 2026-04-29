#!/usr/bin/env bash

set -euo pipefail

assert_single_image_version() {
    local pattern="$1"

    local details
    details="$(git grep -P -o -n "$pattern" | sort -u)"

    local versions
    versions="$(echo "$details" | cut -d: -f3- | cut -d: -f2 | tr -d '[:alpha:]' | sort -uV)"

    local version_count
    version_count="$(echo "$versions" | wc -l)"
    [[ "$version_count" -eq 1 ]] && return 0

    echo "Image version mismatch detected"
    echo
    echo "Found these image references:"
    echo "$details" | cut -d: -f3- | sort -u | while IFS= read -r ref; do
        echo "  - $ref"
    done
    echo
    echo "Locations that need to be updated:"

    while IFS=: read -r file line image_ref; do
        echo "::error file=$file,line=$line:: $file:$line uses '$image_ref'"
    done <<< "$details"

    echo "To fix: Update all files to use the same version."
    echo
    return 1
}

exit_code=0

assert_single_image_version \
    '(quay\.io/stackrox-io/apollo-ci:)?(stackrox|scanner)-(build|test)-[0-9]+\.[0-9]+\.[0-9]+' || exit_code=1

echo

assert_single_image_version \
    '(brew\.registry\.redhat\.io/rh-osbs/)?openshift-golang-builder:.+@sha256:[A-Fa-f0-9]{64}' || exit_code=1

if [[ "$exit_code" -eq 0 ]]; then
    echo "OK"
fi

exit "$exit_code"
