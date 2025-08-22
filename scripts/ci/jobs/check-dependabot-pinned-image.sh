#!/usr/bin/env bash
set -e
updated_dirs=$(yq e '.updates[] | select(.package-ecosystem=="docker") | .directory' .github/dependabot.yaml)

function assert_is_updated() {
    local pinned_file="$1"; shift
    local pinned_dir
    pinned_dir="$(dirname "$pinned_file")"
    for updated_dir in $updated_dirs
    do
        if [[ $pinned_dir = "$updated_dir" ]]
        then
            return 0
        fi
    done
    echo >&2 "File ${pinned_file} is not in a directory in which dependabot updates docker references:"
    echo >&2 "$updated_dirs"
    return 1
}

# shellcheck disable=SC2013
for pinned_file in $(grep -lR PREFETCH-THIS-IMAGE operator/)
do
    assert_is_updated "${pinned_file}"
done
