#!/usr/bin/env bash

# Published images:
# registry.redhat.io/advanced-cluster-security/rhacs
#
# Pre-release images (downstream builds):
# brew.registry.redhat.io/rh-osbs/rhacs
#
# Development images:
# quay.io/rhacs-eng/

set +e -uo pipefail
set -x

GITHUB_STEP_SUMMARY=${GITHUB_STEP_SUMMARY:-/dev/null}

image_prefix="${1:-}"
default_image_prefix='brew.registry.redhat.io/rh-osbs/rhacs'
#default_image_prefix='registry.redhat.io/advanced-cluster-security/rhacs'
image_prefix="${image_prefix:-${default_image_prefix}}"

image_match="${2:-\(bundle\|operator\|rhel8\)$}"
version_filter="${3:-^[0-3]\.}"


function find_images() {
  podman search --limit=100 "${1}" --format "{{.Name}}" \
    | tee >(cat >&2)
}

function latest_tags() {
  while read -r image; do
    if [[ $image != "registry.redhat.io"* ]] && skopeo inspect --override-arch=amd64 --override-os=linux "docker://${image}" > inspect.json; then
      newest_tag=$(jq -r '.RepoTags|.[]' < inspect.json | grep '^[0-9\.\-]*$' | sort -rV | head -1)
    else
      newest_tag=$(podman search --limit=1000000 "${image}" --list-tags --format json \
        | tee inspect.json \
        | jq -r '.[]|.Tags|.[]' | grep '^[0-9\.\-]*$' | sort -rV | head -1)
      skopeo inspect --override-arch=amd64 --override-os=linux "docker://${image}:${newest_tag}" > inspect.json
    fi
    created=$(jq -r '.Created' < inspect.json)
    rm inspect.json
    echo -e "${newest_tag:-latest}\t${image}\t${created}"
  done \
    | tee >(cat >&2) \
    | sort -V
}

function fips_scan() {
  ret=0
  while read -r newest_tag image created; do
    logfile="/tmp/scan-${image##*/}.log"
    ref="${image}${newest_tag:+:}${newest_tag}"
    sha=$(podman pull --arch amd64 --os linux "${ref}")
    if [[ -z "${sha}" ]]; then
      echo "**error:** pulling ${ref}." \
        | tee -a "$GITHUB_STEP_SUMMARY"
      continue
    fi
    if ! podman unshare check-payload \
      scan operator --verbose --spec "${ref}" 2>&1 \
      | tee "${logfile}"; then
      (( ret++ ))  # count images failing fips check
      echo "failed: ${newest_tag:-latest} ${image}@sha256:${sha} ${created}" \
        | tee -a "$GITHUB_STEP_SUMMARY"
    else
      echo "success: ${newest_tag:-latest} ${image}@sha256:${sha} ${created}" \
        | tee -a "$GITHUB_STEP_SUMMARY"
    fi
    for status_type in warning failed; do
      grep --silent "status=\"${status_type}\"" "${logfile}" \
        && echo "${status_type}:" \
        | tee -a "$GITHUB_STEP_SUMMARY"
      grep "status=\"${status_type}\"" "${logfile}" \
        | grep -o 'path=.*error="[^"]*"' \
        | tee -a "$GITHUB_STEP_SUMMARY"
    done
    rm "${logfile}"
  done

  return "${ret}"  # return count of failed images
}


find_images "${image_prefix}" \
  | grep "${image_match}" \
  | latest_tags \
  | grep -v "${version_filter}" \
  | fips_scan
