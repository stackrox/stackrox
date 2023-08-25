#!/usr/bin/env bash

set -euo pipefail

this_file="$(basename "${BASH_SOURCE[0]}")"

usage() {
    >&2 echo "Usage: $this_file <version>

This program prints previous Y-Stream version for a provided <version>.

<version> can be a semantic version, e.g. 3.74.3, and/or the result of make tag, e.g. 3.74.x-nightly-20230224.
<version> can also include 'v' prefix, e.g. v3.74.3.

Y-Stream is Red Hat term for releases which patch number equals to zero, e.g. 3.73.0, 3.74.0, 4.0.0, 4.1.0.
This program knows how to subtract one from the minor number of the provided version (e.g. 3.73.0 -> 3.74.0)
and also knows when major product version was bumped (e.g. 3.74.0 -> 4.0.0)."

    exit 2
}

main() {
    if (( $# > 1 )); then
        >&2 echo "Error: too many command-line arguments provided"
        exit 1
    fi

    local version="${1-}"

    if [[ -z "$version" || "$version" == "--help" ]]; then
        usage
    fi

    if [[ ! "$version" =~ ^v?([0-9]+)\.([0-9]+)\.(x|[0-9]+)(-.+)?$ ]]; then
        >&2 echo "Error: provided version does not look like a valid one: $version"
        exit 1
    fi

    local major="${BASH_REMATCH[1]}"
    local minor="${BASH_REMATCH[2]}"

    print_previous "$major" "$minor"
}

print_previous() {
    local major="$1"
    local minor="$2"

    if (( minor > 0 )); then
        # If the minor version is not zero, than the previous Y-Stream simply had one minor number less.
        echo "$major.$((minor - 1)).0"
    else
        # For major version bumps we need to maintain this mapping of what were previous Y-Streams.
        case "$major" in
        "4") echo "3.74.0" ;;
        "1") echo "0.0.0" ;; # 0.0.0 was never released, but we use 1.0.0 version for "trunk" builds downstream.
        *)
            >&2 echo "Error: don't know the previous Y-Stream for $major.$minor"
            exit 3
        esac
    fi
}

main "$@"
