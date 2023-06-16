#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

GITROOT="$(git rev-parse --show-toplevel)"
[[ -n "${GITROOT}" ]] || { echo >&2 "Could not determine git root!"; exit 1; }

[[ -n "${QUAY_RHACS_ENG_BEARER_TOKEN}" ]] || { echo >&2 "Missing env QUAY_RHACS_ENG_BEARER_TOKEN"; exit 1; }

# Helper method to call curl command to quay
function quay_curl {
    curl -sS --fail -H "Authorization: Bearer ${QUAY_RHACS_ENG_BEARER_TOKEN}" -s -X GET "https://quay.io/api/v1/repository/rhacs-eng/${1}"
}

# Check image scan results in quay.io and alert on new fixable vulns
function compare_fixable_vulns {
  local image_name=$1
  local image_tag=$2

  echo "Fetching current image SHA from quay for $image_name:$image_tag"
  img_data="$(quay_curl "${image_name}/tag/?specificTag=${image_tag}" | jq -r '.tags | first')"
  if [[ "$(jq -r '.is_manifest_list' <<<"$img_data")" == "true" ]]; then
    img_data="$(quay_curl "${image_name}/tag/?specificTag=${image_tag}-amd64" | jq -r '.tags | first')"
  fi
  CURRENT_IMAGE="$(jq -r '.manifest_digest' <<<"$img_data")"
  if [[ -z "$CURRENT_IMAGE" || "$CURRENT_IMAGE" == "null" ]]; then
    echo >&2 "Tag ${image_tag} could not be found for image ${image_name}"
    FAIL_SCRIPT=true
    return
  fi

  # make sure scan is complete before proceeding, since scans would have been started just before running this
  # timeout of 5 mins
  local scan_present
  local count=1

  echo "Getting scan status for ${image_name}"
  wait=30
  count=0
  scan_present=$(quay_curl "${image_name}/manifest/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq -r '.status')
  until [ "$scan_present" = "scanned" ] || [ "$count" -gt 100 ]; do
    echo "${count} Waiting ${wait}s for scan to complete..."
    scan_present=$(quay_curl "${image_name}/manifest/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq -r '.status')
    count=$((count+1))
    sleep $wait
  done

  # if scan never completes, print error message, mark image as failed, and move on to the next
  if [ "$scan_present" != "scanned" ]; then
    echo "${image_name}:${image_tag} scan never completed. Check Quay website."
    FAIL_SCRIPT=true
  else
    echo "Trying to get any fixable vulns for ${image_name}"
    CURRENT_FIXABLE=$(quay_curl "${image_name}/manifest/${CURRENT_IMAGE}/security?vulnerabilities=true" | jq -r '.data.Layer.Features | .[] | select(.Vulnerabilities != null) | .Vulnerabilities | .[] | select(.FixedBy | . != null and . != "") | .Name' | sort -u)

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
RELEASE_TAG=$(make --quiet --no-print-directory -C "${GITROOT}" tag)
COLLECTOR_TAG=$(make --quiet --no-print-directory -C "${GITROOT}" collector-tag)
SCANNER_TAG=$(make --quiet --no-print-directory -C "${GITROOT}" scanner-tag)

ALLOWED_VULNS=$(jq -c '.[]' "$DIR/allowed_vulns.json")

# check main images
compare_fixable_vulns "main" "$RELEASE_TAG"

# check collector images
compare_fixable_vulns "collector" "${COLLECTOR_TAG}-slim"
compare_fixable_vulns "collector" "${COLLECTOR_TAG}"

# check scanner images
compare_fixable_vulns "scanner" "$SCANNER_TAG"
compare_fixable_vulns "scanner-slim" "$SCANNER_TAG"

# check scanner-db images
compare_fixable_vulns "scanner-db" "$SCANNER_TAG"
compare_fixable_vulns "scanner-db-slim" "$SCANNER_TAG"

# if fixable vulns found, return 1 so CI can fail the job
[ "$FAIL_SCRIPT" = true ] && exit 1 || exit 0
