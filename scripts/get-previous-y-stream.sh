#!/usr/bin/env bash

set -euo pipefail

this_file="$(basename "${BASH_SOURCE[0]}")"
this_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bumps_file="${this_dir}/../pkg/version/majorversions/major_version_bumps.yaml"

usage() {
    >&2 echo "Usage: $this_file <version>

This program prints previous Y-Stream version for a provided <version>.

<version> can be a semantic version, e.g. 3.74.3, and/or the result of make tag, e.g. 3.74.x-nightly-20230224.
<version> can also include 'v' prefix, e.g. v3.74.3.

Y-Stream is Red Hat term for releases which patch number equals to zero, e.g. 3.73.0, 3.74.0, 4.0.0, 4.1.0.
This program knows how to subtract one from the minor number of the provided version (e.g. 3.73.0 -> 3.74.0)
and also knows when major product version was bumped (e.g. 3.74.0 -> 4.0.0).

Major version bump history is read from $bumps_file."

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
        echo "$major.$((minor - 1)).0"
        return
    fi

    if [[ ! -f "$bumps_file" ]]; then
        >&2 echo "Error: major version bumps file not found: $bumps_file"
        exit 4
    fi

    # Look up the "from" version for a bump whose "to" matches "$major.0".
    local from
    from=$(awk -v target="$major" '
        /^[[:space:]]*- from:/ { gsub(/[" ]/, "", $0); split($0, a, ":"); from_val = a[2] }
        /^[[:space:]]*to:/ { gsub(/[" ]/, "", $0); split($0, a, ":"); to_val = a[2];
            split(to_val, parts, ".");
            if (parts[1] == target && parts[2] == "0") { print from_val; exit }
        }
    ' "$bumps_file")

    if [[ -z "$from" ]]; then
        >&2 echo "Error: don't know the previous Y-Stream for $major.$minor"
        exit 3
    fi

    echo "${from}.0"
}

main "$@"
