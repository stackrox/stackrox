#!/usr/bin/env bash

# Emits `--set manager.envOverrides.RELATED_IMAGE_*=...` Helm arguments that point
# every related image the operator knows about at the corresponding Konflux release
# image for a given version. Used by the `deploy-via-chart` target when
# USE_KONFLUX_IMAGES=true.
#
# The list of related images is derived from the single source of truth -
# the manager Deployment env in operator/config/manager/manager.yaml - so that
# adding (or removing) a RELATED_IMAGE_* there automatically updates the Helm
# arguments, and the two cannot silently drift apart.
#
# Image repository names default to the env var name convention
# (RELATED_IMAGE_SCANNER_V4_DB -> scanner-v4-db) and can be overridden per entry
# with an `image-name=<name>` tag in the trailing `## tags:` comment.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly ROOT_DIR

# Konflux release images all live under this registry/prefix, tagged with the version.
readonly REGISTRY="quay.io/rhacs-eng"
readonly REPO_PREFIX="release-"

readonly MANAGER_YAML="${ROOT_DIR}/operator/config/manager/manager.yaml"

VERSION="${1:-${VERSION:-}}"
if [[ -z "${VERSION}" ]]; then
    echo "usage: $0 <version> (or set VERSION env var)" >&2
    exit 1
fi
readonly VERSION

args=()
while IFS= read -r line; do
    # Example lines being matched:
    #   - name: RELATED_IMAGE_MAIN ## tags: uses-main-tag,image-name=main
    #   - name: RELATED_IMAGE_COLLECTOR
    var_name="$(sed -Ee 's/^[[:space:]]*-[[:space:]]*name:[[:space:]]*([A-Za-z0-9_]+).*/\1/' <<< "${line}")"

    # Derive the image repository name from the var name by convention, then let an
    # explicit image-name tag override it.
    image_name="$(tr '[:upper:]_' '[:lower:]-' <<< "${var_name#RELATED_IMAGE_}")"
    if [[ "${line}" == *"image-name="* ]]; then
        image_name="$(sed -Ee 's/.*image-name=([^, ]+).*/\1/' <<< "${line}")"
    fi

    args+=("--set" "manager.envOverrides.${var_name}=${REGISTRY}/${REPO_PREFIX}${image_name}:${VERSION}")
done < <(grep -E '^[[:space:]]*-[[:space:]]*name:[[:space:]]*RELATED_IMAGE_' "${MANAGER_YAML}")

if [[ ${#args[@]} -eq 0 ]]; then
    echo "error: no RELATED_IMAGE_* entries found in ${MANAGER_YAML}" >&2
    exit 1
fi

echo "${args[@]}"
