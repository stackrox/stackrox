#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

GITROOT="$(git rev-parse --show-toplevel)"
[[ -n "${GITROOT}" ]] || { echo >&2 "Could not determine git root!"; exit 1; }

[[ -n "${QUAY_BEARER_TOKEN}" ]] || { echo >&2 "Missing env QUAY_BEARER_TOKEN"; exit 1; }

# Helper method to call curl command to quay
function quay_curl {
    curl -sS --fail -H "Authorization: Bearer ${QUAY_BEARER_TOKEN}" -s -X GET "https://quay.io/api/v1/repository/stackrox/${1}"
}

# Check image scan results in quay.io and alert on new fixable vulns
function compare_fixable_vulns {
  local image_name=$1
  local image_tag=$2

  echo "Fetching current image id from quay for $image_name:$image_tag"
  CURRENT_IMAGE="$(quay_curl "${image_name}/tag/" | jq --arg CURRENT_TAG "${image_tag}" '.tags | first(.[] | select(.name==$CURRENT_TAG)) | .image_id' | tr -d '\"')"

  # make sure scan is complete before proceeding, since scans would have been started just before running this
  # timeout of 5 mins
  local scan_present
  local count=1

  echo "Getting scan status"
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
    echo "Trying to get any fixable vulns for the scanned image"
    CURRENT_FIXABLE=$(quay_curl "${image_name}/image/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq -r '.data.Layer.Features | .[] | select(.Vulnerabilities != null) | .Vulnerabilities | .[] | select(.FixedBy != null) | .Name')

    # fail the check if fixable vulns are found that are not allowed
    if [[ -n "$CURRENT_FIXABLE" ]]; then
      echo "${image_name}:${image_tag} has fixable vulns!:"
          IFS='
'
      for vuln in $CURRENT_FIXABLE; do
        is_allowed=0
        for allowed in $ALLOWED_VULNS; do
          allowed_vuln=$(echo "$allowed" | jq -r '.vuln')
          allowed_image=$(echo "$allowed" | jq -r '.image')
          allowed_tag=$(echo "$allowed" | jq -r '.tag')
          if [[ "${vuln}" == ${allowed_vuln} || "$allowed_vuln" == "*" ]] &&
             [[ "${image_name}" =~ ${allowed_image} ]] &&
             [[ "${image_tag}" =~ ${allowed_tag} ]]
          then
            echo "  Allowing ${vuln} because it matches ${allowed}."
            is_allowed=1
            break
          fi
        done
        if (( ! is_allowed )); then
          FAIL_SCRIPT=true
          echo "  ${vuln} is fixable and not in allowed_vulns.json"
        fi
      done
    else
      echo "${image_name}:${image_tag} has no fixable vulns"
    fi
  fi
}

FAIL_SCRIPT=false

# determine all image tags
RELEASE_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" tag)
COLLECTOR_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" collector-tag)
SCANNER_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" scanner-tag)
DOCS_PRERELEASE_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" docs-tag)

ALLOWED_VULNS=$(jq -c '.[]' "$DIR/allowed_vulns.json")

# check main images
compare_fixable_vulns "main" "$RELEASE_TAG"
compare_fixable_vulns "main-rhel" "$RELEASE_TAG"

# check docs image - using the pre-release tag (not the release tag)
compare_fixable_vulns "docs" "$DOCS_PRERELEASE_TAG"

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
