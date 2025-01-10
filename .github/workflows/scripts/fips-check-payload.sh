#!/usr/bin/env bash

# Published images:
# registry.redhat.io/advanced-cluster-security/rhacs
#
# Staged images (downstream builds):
# brew.registry.redhat.io/rh-osbs/rhacs
#
# Development images:
# quay.io/rhacs-eng/

set +e -uo pipefail

GITHUB_STEP_SUMMARY=${GITHUB_STEP_SUMMARY:-/dev/null}

image_prefix="${1:-}"
default_image_prefix='brew.registry.redhat.io/rh-osbs/rhacs'
image_prefix="${image_prefix:-${default_image_prefix}}"

function latest_tags() {
  for image in $(podman search --limit=100 "${image_prefix}" --format "{{.Name}}"); do
    skopeo inspect --override-arch=amd64 --override-os=linux "docker://${image}" > inspect.json
    newest_tag=$(jq -r '.RepoTags|.[]' < inspect.json | grep '^[0-9\.\-]*$' | sort -rV | head -1)
    created=$(jq -r '.Created' < inspect.json)
    rm inspect.json
    echo -e "${newest_tag}\t${image}\t${created}"
  done \
    | sort -V
}

function fips_scan() {
  ret=0
  while read -r newest_tag image created; do
    logfile="/tmp/scan-${image##*/}.log"
    ref="${image}${newest_tag:+:}${newest_tag}"
    sha=$(podman pull --arch amd64 --os linux "${ref}")
    echo "${newest_tag:-latest} ${image}@sha256:${sha} (created:${created})" \
      | tee -a "$GITHUB_STEP_SUMMARY"
    if ! podman unshare check-payload \
      scan operator --verbose --spec "${ref}" 2>&1 \
      | tee "${logfile}"; then
      (( ret++ ))  # count images failing fips check
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

latest_tags "${image_prefix}" \
  | fips_scan
