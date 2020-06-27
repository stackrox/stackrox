#!/usr/bin/env bash

# Monitor stdin and echo to stdout. If nothing is read for a given interval
# issue a warning. Exit when stdin closes.

usage() {
    echo "$0 <output delay>" >&2
    exit 1
}

main() {
    if [[ "$#" -ne 1 ]]; then
        usage
    fi

    timeout="$1"

    while true; do
        read -t "$timeout" -r line
        ret="$?"

        if [[ "$ret" -eq 0 ]]; then
            echo "$line"
        elif [[ "$ret" -gt 128 ]]; then
            echo "$(date) - No output in ${timeout} seconds"
        else
            break
        fi
    done
}

main "$@"
