#!/usr/bin/env bash

set -euo pipefail
# shellcheck source=./common.sh
source "$(dirname "$0")/common.sh"

if [[ -z "${MAIN_IMAGE_TAG:-}" ]]; then
    MAIN_IMAGE_TAG="$(make -sC "$ROOT_DIR" tag)"
fi

IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"
if [[ -n "$IMAGE_REGISTRY" ]]; then
    IMAGE_REGISTRY="$(echo "$IMAGE_REGISTRY" | sed -Ee 's|^(.*)/?$|\1/|')"
fi

# Inspects the magic tags on the related image environment variables in
# operator/bundle/manifests/rhacs-operator.clusterserviceversion.yaml and
# extracts those related images that have the "uses-main-tag" tag set to true.
#
# Respects MAIN_IMAGE_TAG and IMAGE_REGISTRY environment variables.
function extract_related_images_with_main_tags() {
    local deployment="rhacs-operator-controller-manager"
    local container="manager"
    cat "${ROOT_DIR}/operator/bundle/manifests/rhacs-operator.clusterserviceversion.yaml" \
        | yq e ".spec.install.spec.deployments[] \
                | select(.name == \"$deployment\") \
                | .spec.template.spec.containers[] \
                | select(.name == \"$container\") \
                | .env[] \
                | select(.name | test(\"^RELATED_\"))" \
        | grep '## tags:' \
        | sed -e 's/name: //;' \
        | sort \
        | while read -r line; do
            local var_name
            var_name="$(echo "$line" | cut -d' ' -f1)"
            tags_info="$(echo "$line" | sed -Ee 's/.* ## tags: *(.*)$/\1/;')"
            if [[ "$(extract_tag "$tags_info" "uses-main-tag")" == "true" ]]; then
                local image_name
                image_name="$(extract_tag "$tags_info" "image-name")"
                echo "${var_name}=${IMAGE_REGISTRY}${image_name}:${MAIN_IMAGE_TAG}"
            fi
        done
}

# Extracts the value of a specific tag from the tags info string.
# If the tag is present without a value, returns "true".
# If the tag is absent, returns "false".
# If the tag is present with an assigned value, returns the value.
#
# Arguments:
#   $1 - The tags info string (comma-separated key-value pairs).
#   $2 - The tag key to extract.
# Returns:
#   The value of the specified tag, "true", or "false".
function extract_tag() {
    local tags_info="$1"
    local tag_key="$2"
    local tag_value=""
    local tag_found
    tag_found="$(echo "$tags_info" | tr ',' '\n' | grep -E " *${tag_key}(=[^ ]*)? *")"
    local tag_value="false"
    if [[ -n "$tag_found" ]]; then
        tag_value="true"
        if [[ "$tag_found" == *"="* ]]; then
            tag_value="$(echo "$tag_found" | cut -d'=' -f2)"
        fi
    fi
    echo "${tag_value}"
}

extract_related_images_with_main_tags
