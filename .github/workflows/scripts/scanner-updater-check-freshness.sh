#!/usr/bin/env bash
#
# Check whether a GCS object is newer than a reference point.
#
# Uses an unauthenticated curl against the public GCS download URL to
# classify the object state (200/404/other), then gcloud for the object
# timestamp when it exists.
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
# Exit codes:
#   0 — object exists and is newer than the threshold
#   1 — object is missing (404) or not newer than the threshold
#   2 — unexpected failure
#
# Inputs (environment):
#   GCS_OBJECT — full GCS object path (gs://bucket/path/to/object)
#
# Requires: curl, gcloud, date.

set -euo pipefail

# The ERR trap below forces any unexpected command failure to exit 2 instead of
# the failing command's own exit code. Without it, set -e would exit with the
# failing command's code (usually 1), making it indistinguishable from the
# intentional "not fresh" exit. The trap guarantees that exit 1 can only come
# from our explicit exit 1 calls.
trap 'echo "::error::unexpected failure in check-freshness.sh at line $LINENO"; exit 2' ERR

threshold=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --newer-than)
            threshold="${2:?--newer-than requires an epoch value}"
            shift 2
            ;;
        --max-age)
            max_age="${2:?--max-age requires a value in seconds}"
            threshold=$(( $(date +%s) - max_age ))
            shift 2
            ;;
        *)
            echo "::error::unknown argument: $1"
            exit 2
            ;;
    esac
done

if [[ -z "$threshold" ]]; then
    echo "::error::one of --newer-than or --max-age is required"
    exit 2
fi

object="${GCS_OBJECT:?GCS_OBJECT is required}"

# Derive the download URL from the gs:// path.
download_url="https://storage.googleapis.com/${object#gs://}"

# Check object existence via unauthenticated curl.
http_code=$(curl -s -o /dev/null -w "%{http_code}" \
    --connect-timeout 10 --max-time 15 \
    "$download_url")

case "$http_code" in
    200)
        creation_time=$(gcloud storage objects describe "$object" \
            --format="value(creation_time)")
        object_epoch=$(date -d "$creation_time" +%s)
        if (( object_epoch > threshold )); then
            echo >&2 "INFO: ${object}: newer than threshold (${object_epoch} > ${threshold})"
            exit 0
        fi
        echo >&2 "INFO: ${object}: not newer than threshold (${object_epoch} <= ${threshold})"
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
