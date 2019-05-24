#!/usr/bin/env bash
# staticcheck wrapper that ignores certain failures that we are okay with.

set -e

ignored=0
total=0

[[ -x "$(command -v staticcheck)" ]] || { echo >&2 "staticcheck binary not found in path!"; exit 1; }

while read -r line; do
    total=$((total + 1))
    if [[ "$line" =~ ^.*var\ log\ is\ unused ]]; then
        ignored=$((ignored + 1))
    else
        echo >&2 "${line}"
    fi
done < <(staticcheck -checks=all,-ST1000,-ST1001,-ST1005,-SA1019,-SA4001,-ST1016 "$@")

echo "Found ${total} errors, ignored ${ignored}"
if (( total == ignored )); then
    exit 0
else
    exit 1
fi
