#!/usr/bin/env bash
set -euo pipefail
# Reads a list of files (one per line) from stdin and prints on stdout a list of go packages.
# The arguments are passed to `go list` that is used to perform the transformation.
# That go list command is retried, to help with go proxy unavailability.

num_retries=4

dir_list=$(mktemp)
sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq > "${dir_list}"
output=$(mktemp)
for attempt in $(seq 0 $num_retries)
do
    sleep_sec=$((attempt * 5))
    if [[ $sleep_sec -gt 0 ]]; then
        echo >&2 "Sleeping ${sleep_sec} before retrying..."
        sleep "${sleep_sec}"
    fi
    if xargs go list "$@" < "${dir_list}" > "${output}"; then
        grep -v '^github.com/stackrox/rox/tests$' "${output}"
        exit 0
    fi
done
echo >&2 "Running 'go list $*' failed despite $num_retries retries."
exit 1
