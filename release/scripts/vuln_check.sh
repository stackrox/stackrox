#!/usr/bin/env bash

set -eu

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Helper method to call curl command to quay
function quay_curl {
    curl -H "Authorization: Bearer ${QUAY_BEARER_TOKEN}" -s -X GET "https://quay.io/api/v1/repository/stackrox/${1}"
}

# Check image scan results in quay.io and alert on new fixable vulns
function compare_fixable_vulns {
  local image_name=$1
  local image_tag=$2

  # fetch current image id from quay
  CURRENT_IMAGE="$(quay_curl "${image_name}/tag/" | jq --arg CURRENT_TAG "${image_tag}" '.tags | first(.[] | select(.name==$CURRENT_TAG)) | .image_id' | tr -d '\"')"

  # make sure scan is complete before proceeding, since scans would have been started just before running this
  # timeout of 5 mins
  local scan_present
  local count=1

  scan_present=$(quay_curl "${image_name}/image/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq '.status')
  until [ "$(echo "$scan_present" | tr -d '\"')" = "scanned" ] || [ "$count" -gt 60 ]; do
    echo "Waiting for scan to complete..."
    scan_present=$(quay_curl "${image_name}/image/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq '.status')
    count=$((count+1))
    sleep 15
  done

  # if scan never completes, print error message, mark image as failed, and move on to the next
  if [ "$(echo "$scan_present" | tr -d '\"')" != "scanned" ]; then
    echo "${image_name}:${image_tag} scan never completed. Check Quay website."
    FAIL_SCRIPT=true
  else
    # get any fixable vulns for the scanned image
    CURRENT_FIXABLE=$(quay_curl "${image_name}/image/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq '.data.Layer.Features | .[] | select(.Vulnerabilities != null) | .Vulnerabilities | .[] | select(.FixedBy != null) | .Name')

    # if fixabnle vulns found, print them out and set script to return error status
    if [[ -n "$CURRENT_FIXABLE" ]]; then
      FAIL_SCRIPT=true
      echo "${image_name}:${image_tag} has fixable vulns!:"
      echo "$CURRENT_FIXABLE"
    else
      echo "${image_name}:${image_tag} has no fixable vulns"
    fi
  fi
}

FAIL_SCRIPT=false

# determine all image tags
RELEASE_TAG=$(git describe --tags)
COLLECTOR_TAG=$(cat "$DIR/../../COLLECTOR_VERSION")
SCANNER_TAG=$(cat "$DIR/../../SCANNER_VERSION")

# check main images
compare_fixable_vulns "main" "$RELEASE_TAG"
compare_fixable_vulns "main-rhel" "$RELEASE_TAG"

# check monitoring image - skip for now because we don't really care :(
# compare_fixable_vulns "monitoring" "$RELEASE_TAG"

# check collector images
compare_fixable_vulns "collector" "$COLLECTOR_TAG"
compare_fixable_vulns "collector-rhel" "$COLLECTOR_TAG"

# check scanner images
compare_fixable_vulns "scanner" "$SCANNER_TAG"
compare_fixable_vulns "scanner-rhel" "$SCANNER_TAG"

# check scanner-db images
compare_fixable_vulns "scanner-db" "$SCANNER_TAG"
compare_fixable_vulns "scanner-db-rhel" "$SCANNER_TAG"

# if fixable vulns found, return 1 so CI can fail the job
[ "$FAIL_SCRIPT" = true ] && exit 1 || exit 0
