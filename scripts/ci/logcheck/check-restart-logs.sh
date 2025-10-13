#!/usr/bin/env bash

# Checks if log files from pod restarts have patterns that indicate the restart is ok.
# It returns a zero exit status if all log files have an ok indicating pattern.

set -e

DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ -z "$*" || $# -lt 2 ]]; then
    echo "Usage: check-restart-logs.sh <CI job> <files...>"
    exit 1
fi

job=$1
shift

for logfile in "$@"; do
    if [[ ! -f "${logfile}" ]]; then
        echo "Error: the log file '${logfile}' does not exist"
        exit 1
    fi
done

patterns=$(jq -c '.[]' "$DIR/restart-ok-patterns.json")
(
    IFS='
'
    all_ok=true
    for logfile in "$@"; do
        echo "Checking for a restart exception in: ${logfile}"
        this_log_is_ok=false
        for pattern in $patterns; do
            comment=$(echo "$pattern" | jq -r '.comment')
            job_pattern=$(echo "$pattern" | jq -r '.job')
            logfile_pattern=$(echo "$pattern" | jq -r '.logfile')
            logline_pattern=$(echo "$pattern" | jq -r '.logline')
            if [[ "${job}" =~ ${job_pattern} ]] &&
               [[ "${logfile}" =~ ${logfile_pattern} ]] &&
               egrep -q "${logline_pattern}" "${logfile}"
            then
                echo "Ignoring this restart due to: ${comment}"
                this_log_is_ok=true
                break
            fi
        done
        if ! ${this_log_is_ok}; then
            echo "This restart does not match any ignore patterns"
            if [[ -n "${ARTIFACT_DIR:-}" ]]; then
                cp "${logfile}" "${ARTIFACT_DIR}" || true
                echo "$(basename "${logfile}") copied to Artifacts" # do not change - required by pod restart check
            fi
            all_ok=false
        fi
    done
    if ! ${all_ok}; then
        exit 2
    fi
)

exit 0
