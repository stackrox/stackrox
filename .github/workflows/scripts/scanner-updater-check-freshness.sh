#!/usr/bin/env bash
#
# Check whether an updater's vulnerability data is fresh.
#
# Uses an unauthenticated curl against the public GCS download URL to
# classify the object state (200/404/other), then gcloud for creation_time
# when the object exists.
#
# Exit codes:
#   0 — object exists and is fresh (age < interval)
#   1 — object is missing (404) or stale (age >= interval)
#   2 — unexpected failure: caller should treat as error and notify
#
# Inputs (environment):
#   GCS_OBJECT       — full GCS object path (gs://bucket/path/to/data.json.zst)
#   UPDATER_SOURCE   — updater name, used to look up interval in config
#   SOURCE_CONFIG    — path to source-config.yaml
#
# Requires: curl, yq, gcloud.

set -euo pipefail

# The ERR trap below forces any unexpected command failure to exit 2 instead of
# the failing command's own exit code. Without it, set -e would exit with the
# failing command's code (usually 1), making it indistinguishable from the
# intentional "not fresh" exit. The trap guarantees that exit 1 can only come
# from our explicit exit 1 calls.
trap 'echo "::error::unexpected failure in check-freshness.sh at line $LINENO"; exit 2' ERR

object="${GCS_OBJECT:?GCS_OBJECT is required}"
updater="${UPDATER_SOURCE:?UPDATER_SOURCE is required}"
config="${SOURCE_CONFIG:?SOURCE_CONFIG is required}"

# Derive the download URL from the gs:// path.
download_url="https://storage.googleapis.com/${object#gs://}"

duration_to_seconds() {
    local dur="${1:?}"
    local total=0 remaining="$dur"
    while [[ -n "$remaining" ]]; do
        if [[ "$remaining" =~ ^([0-9]+)h(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] * 3600 )); remaining="${BASH_REMATCH[2]}"
        elif [[ "$remaining" =~ ^([0-9]+)m(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] * 60 )); remaining="${BASH_REMATCH[2]}"
        elif [[ "$remaining" =~ ^([0-9]+)s(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] )); remaining="${BASH_REMATCH[2]}"
        else
            echo "::error::cannot parse duration: $remaining"; exit 2
        fi
    done
    echo "$total"
}

# Read interval from config.
default_interval=$(yq '.updaters.default.interval // "4h"' "$config")
interval=$(yq ".updaters.\"${updater}\".interval // \"$default_interval\"" "$config")
interval_secs=$(duration_to_seconds "$interval")

# Check object existence via unauthenticated curl.
http_code=$(curl -s -o /dev/null -w "%{http_code}" \
    --connect-timeout 10 --max-time 15 \
    "$download_url")

case "$http_code" in
    200)
        creation_time=$(gcloud storage objects describe "$object" \
            --format="value(creation_time)")
        created_epoch=$(date -d "$creation_time" +%s)
        now=$(date +%s)
        age=$(( now - created_epoch ))
        if (( age < interval_secs )); then
            echo >&2 "INFO: ${object}: fresh (age=${age}s < interval=${interval_secs}s)"
            exit 0
        fi
        echo >&2 "INFO: ${object}: stale (age=${age}s >= interval=${interval_secs}s)"
        exit 1
        ;;
    404)
        echo >&2 "INFO: ${object}: not found"
        exit 1
        ;;
    *)
        echo "::error::${object}: GCS returned HTTP $http_code"
        exit 2
        ;;
esac
