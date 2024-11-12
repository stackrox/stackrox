#!/usr/bin/env bash
#
# CAUTION: If you change this file, please be sure to update the corresponding midstream script
#
# Downloads and performs basic JSON validation for the name-to-repository and
# repository-to-cpu mapping files to be embedded in the Scanner v4 container.

set -euo pipefail

main() {
    local output_dir="$1"

    local urls=(
        "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
        "https://security.access.redhat.com/data/metrics/container-name-repos-map.json"
    )

    echo "Downloading mapping files"

    local url
    for url in "${urls[@]}"; do
        local filename
        filename=$(basename "$url")
        echo "Downloading ${url} > ${output_dir}/$filename"
        curl --location --silent --fail --show-error --retry 3 \
            -o "${output_dir}/$filename" "$url"
        if [[ ! -s "${output_dir}/$filename" ]]; then
            echo "${output_dir}/$filename is empty"
            exit 1
        fi

        if ! check_valid_json "${output_dir}/$filename"; then
            echo "${output_dir}/$filename is invalid JSON"
            exit 1
        fi
    done

    echo "Done"
}

# check_valid_json(filename)
#
# Validates that the contents of the specified file is JSON.
#
# Arguments:
#   filename: Name of the file whose contents should be validated
check_valid_json() {
    local filename="$1"

    echo "Validating if $filename is valid JSON"
    if command -v jq &>/dev/null; then
        echo "Using jq"
        if ! jq -e . >/dev/null 2>&1 < "$filename"; then
            echo "$filename is not valid JSON"
            exit 1
        fi
    elif command -v python &>/dev/null; then
        echo "Using python"
        if ! python -c "import json; f = open('$filename'); json.load(f); f.close()"; then
            echo "$filename is not valid JSON"
            exit 1
        fi
    else
        echo "WARNING: could not detect a method to validate JSON"
    fi

    echo "Valid!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -ne "1" ]]; then
        >&2 echo "Usage: $0 <target directory>"
        exit 1
    fi

    main "$@"
fi
