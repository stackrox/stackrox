#!/usr/bin/env bash
set -euo pipefail

# Use:
# ./metrics-merge-yaml.sh <list of YAML files> <output file>

# Example:
# ./metrics-merge-yaml.sh metrics-base.yaml metrics-central.yaml metrics-sensor.yaml metrics-collector.yaml metrics.yaml

main() {
    if (( "$#" < 2 )); then
        echo "Usage: ./metrics-merge-yaml.sh <list of YAML files> <output file>" >&2
        exit 1
    fi

    local args=( "$@" )
    local args_last_index=$(("${#args[@]}"-1))
    local output_file_name="${args[args_last_index]}"

    for ((i=0; i < args_last_index; i++)); do
        [[ -f "${args[$i]}" ]] || { echo "Input file not found: ${args[$i]}" >&2; exit 1; }
    done

    echo "Merge all files into: ${output_file_name}"

    truncate -s 0 "${output_file_name}"
    for ((i=0; i < args_last_index; i++))
    do
        { echo -e ""; echo -e "# File: ${args[$i]}"; echo -e ""; cat "${args[$i]}"; } >> "${output_file_name}"
    done

    echo "Done!"
}

main "$@"
