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

#image_prefix="${1:-brew.registry.redhat.io/rh-osbs/rhacs-main-rhel8}"
#image_prefix="${1:-registry.redhat.io/advanced-cluster-security/rhacs-}"
#default_image_prefix='brew.registry.redhat.io/rh-osbs/rhacs'
default_image_prefix='registry.redhat.io/advanced-cluster-security/rhacs'
image_prefix="${image_prefix:-${default_image_prefix}}"

image_match="${2:-\(bundle\|operator\|rhel8\|stackrox\)$}"
#image_match="${2:-\(bundle\|operator\|\(roxctl\|slim\|db\|v4\|scanner\|main\|collector\)\(-rhel.\)\?\)$}"
#image_match="${2:-\(bundle\|operator\|\(roxctl\|slim\|db\|v4\|scanner\|main\|collector\)-rhel8\|stackrox\)$}"
image_filter="${3:-drivers}"
version_match="${4:-^[^0-3]\.}"


function find_images() {
  podman search --limit=100 "${1}" --format "{{.Name}}" \
    | tee >(cat >&2)
}

function latest_tags() {
  local version_match=${1:-^[^0-3]\.}

  while read -r image; do
    if [[ $image != "registry.redhat.io"* ]] && skopeo inspect --override-arch=amd64 --override-os=linux "docker://${image}" > inspect.json; then
      newest_tag=$(jq -r '.RepoTags|.[]' < inspect.json \
        | grep '^[0-9\.\-]*$' \
        | grep "${version_match}" \
        | sort -rV \
        | head -1)
    else
      newest_tag=$(podman search --limit=1000000 "${image}" --list-tags --format json \
        | tee inspect.json \
        | jq -r '.[]|.Tags|.[]' \
        | grep '^[0-9\.\-]*$' \
        | grep "${version_match}" \
        | sort -rV \
        | head -1)
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

while read -r ref; do
  logfile="/tmp/scan-${ref##*/}.log"
  sha=$(podman pull --arch amd64 --os linux "${ref}")
  if ! podman unshare check-payload \
    scan operator --verbose --spec "${ref}" 2>&1 \
    | tee "${logfile}"; then
    echo "failed: ${ref} ${sha}" \
      | tee -a "$GITHUB_STEP_SUMMARY"
  else
    echo "success: ${ref} ${sha}" \
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
done <<EOF
registry.redhat.io/advanced-cluster-security/rhacs-central-db-rhel8@sha256:140a967924f794964cf565fed54f53f1fdf54866c26064a98c624a9a7d47e190
registry.redhat.io/advanced-cluster-security/rhacs-collector-rhel8@sha256:b90dc550d570a4ff508cc028076b676a8bfb99363170fc3f9c58878bc8956a38
registry.redhat.io/advanced-cluster-security/rhacs-main-rhel8@sha256:c27e69bc48c16ae3c7bd53f8b956a12669b1de9f8a0fc31ff7c2ac198c09dfb8
registry.redhat.io/advanced-cluster-security/rhacs-rhel8-operator@sha256:3eb6b6747257dcf216739a1eb7157b72d0ee6ffa3b1d76c5effe62bbe97b5fb5
registry.redhat.io/advanced-cluster-security/rhacs-operator-bundle@sha256:44823edf63a673d6df937271c2b504782752e832d31517076593e1706a6cf434
registry.redhat.io/advanced-cluster-security/rhacs-roxctl-rhel8@sha256:b9ae54b5b98285b140ffddcc105db8531e6c28a58eebdfcac9af54e24d0c94f1
registry.redhat.io/advanced-cluster-security/rhacs-scanner-rhel8@sha256:7c8fcbbb09cdc07ad92b20cec71175b9f234b65c1b3220e1e48084b23d28aaa3
registry.redhat.io/advanced-cluster-security/rhacs-scanner-db-rhel8@sha256:4a6f305e503999bbd2d0c3fb66a177779ba8e1882d3643d5d3e0a508f3a5f396
registry.redhat.io/advanced-cluster-security/rhacs-scanner-db-slim-rhel8@sha256:ead12fce0b0f032b633e4c98f3acaf641fdfaefe3c883eb3d06a1c95c6063b9f
registry.redhat.io/advanced-cluster-security/rhacs-scanner-slim-rhel8@sha256:2799fa8b5d4862e7b970ff9187024ddbfb64b183d0df72cae53e2bde68996684
registry.redhat.io/advanced-cluster-security/rhacs-scanner-v4-rhel8@sha256:f200ed14ea58040d41d5017190d312bf60fb5dcd6c6c423bad01854cb3e9e576
registry.redhat.io/advanced-cluster-security/rhacs-scanner-v4-db-rhel8@sha256:8f78565f7d2051bd9be4d02b0910d1cccbd6122967ad87aa63b8f6e91bc121df
EOF

exit 0
find_images "${image_prefix}" \
  | grep "${image_match}" \
  | grep -v "${image_filter}" \
  | latest_tags "${version_match}" \
  | fips_scan
