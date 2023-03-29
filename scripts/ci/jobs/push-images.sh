#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

push_images() {
    info "Will push images built in CI"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: push_images <brand>"
    fi

    local brand="$1"

    info "Images from OpenShift CI builds:"
    env | grep IMAGE || true

    [[ "${OPENSHIFT_CI:-false}" == "true" ]] || { die "Only supported in OpenShift CI"; }

    local tag
    tag="$(make --quiet tag)"

    if is_release_version "$tag"; then
        check_scanner_and_collector_versions
    else
        info "Not checking version files for non releases"
    fi

    local push_context=""
    local base_ref
    base_ref="$(get_base_ref)" || {
        info "Warning: error running get_base_ref():"
        echo "${base_ref}"
        info "will continue with pushing images."
    }
    if ! is_in_PR_context && [[ "${base_ref}" == "master" ]]; then
        push_context="merge-to-master"
    fi

    if [[ -n "${MAIN_IMAGE}" ]]; then
        if is_OPENSHIFT_CI && [[ "$brand" == "RHACS_BRANDING" ]]; then
            info "Images were built and pushed elsewhere, skipping it here."
        else
            push_main_image_set "$push_context" "$brand"
            push_matching_collector_scanner_images "$brand"
        fi
    fi
    if [[ -n "${MAIN_RCD_IMAGE:-}" ]]; then
        if is_OPENSHIFT_CI; then
            info "-race image was built and pushed elsewhere, skipping it here."
        else
            push_race_condition_debug_image
        fi
    fi
    if [[ -n "${OPERATOR_IMAGE:-}" ]]; then
        if is_OPENSHIFT_CI; then
            info "Operator images were built and pushed elsewhere, skipping it here."
        else
            push_operator_image_set "$push_context" "$brand"
        fi
    fi
    if [[ -n "${MOCK_GRPC_SERVER_IMAGE:-}" ]]; then
        if is_OPENSHIFT_CI; then
            info "Mock GRPC image was built and pushed elsewhere, skipping it here."
        else
            push_mock_grpc_server_image
        fi
    fi
}

push_images "$@"
