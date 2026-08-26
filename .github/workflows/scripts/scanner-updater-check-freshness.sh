#!/usr/bin/env bash
#
# Check whether a GCS object is newer than a reference point.
#
# Uses an unauthenticated HTTP HEAD request against the public GCS
# download URL to classify the object state (200/404/other), then
# gcloud for the object timestamp when it exists.
#
# Exactly one of --newer-than or --max-age must be provided.
#
# --newer-than <epoch>:
#   The object is "fresh" if its timestamp is after <epoch>.
#
# --max-age <seconds>:
#   The object is "fresh" if its age is below <seconds>
#   (equivalent to --newer-than "$(date +%s) - N").
#
# Output (stdout):
#   "true"  — object exists and is newer than the threshold
#   "false" — object is not newer, or does not exist (404)
#
# Exit codes:
#   0 — check completed (result is on stdout)
#   non-zero — unexpected failure (stdout is irrelevant)
#
# Inputs (environment):
#   GCS_OBJECT — full GCS object path (gs://bucket/path/to/object)
#
# Requires: curl, gcloud, date.

set -euo pipefail

trap 'echo "::error::unexpected failure in check-freshness.sh at line $LINENO" >&2; exit 2' ERR

threshold=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --newer-than)
            if [[ -n "$threshold" ]]; then
                echo "::error::--newer-than and --max-age are mutually exclusive" >&2
                exit 1
            fi
            threshold="${2:?--newer-than requires an epoch value}"
            shift 2
            ;;
        --max-age)
            if [[ -n "$threshold" ]]; then
                echo "::error::--newer-than and --max-age are mutually exclusive" >&2
                exit 1
            fi
            max_age="${2:?--max-age requires a value in seconds}"
            threshold=$(( $(date +%s) - max_age ))
            shift 2
            ;;
        *)
            echo "::error::unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if [[ -z "$threshold" ]]; then
    echo "::error::one of --newer-than or --max-age is required" >&2
    exit 1
fi

object="${GCS_OBJECT:?GCS_OBJECT is required}"

# Derive the public download URL from the gs:// path.
download_url="https://storage.googleapis.com/${object#gs://}"

http_code=$(curl --head --silent --show-error -o /dev/null -w "%{http_code}" \
    --connect-timeout 10 --max-time 15 \
    "$download_url")

case "$http_code" in
    200)
        creation_time=$(timeout 30 gcloud storage objects describe "$object" \
            --format="value(creation_time)")
        object_epoch=$(date -d "$creation_time" +%s)
        if (( object_epoch > threshold )); then
            echo >&2 "INFO: ${object}: newer than threshold (${object_epoch} > ${threshold})"
            echo "true"
        else
            echo >&2 "INFO: ${object}: not newer than threshold (${object_epoch} <= ${threshold})"
            echo "false"
        fi
        ;;
    404)
        echo >&2 "INFO: ${object}: not found"
        echo "false"
        ;;
    *)
        echo "::error::${object}: GCS returned HTTP $http_code" >&2
        exit 1
        ;;
esac
