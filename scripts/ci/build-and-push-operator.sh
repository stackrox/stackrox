#!/usr/bin/env bash

set -euo pipefail

# Build and Push Operator Images
#
# Builds operator images for all brandings and architectures, then creates
# and pushes multi-arch manifests. Replaces the old matrix-based workflow
# with a simpler sequential approach.
#
# Usage:
#   build-and-push-operator.sh <github_event_name> <github_ref_name> <archs>
#
# Example:
#   ./scripts/ci/build-and-push-operator.sh pull_request main "amd64 arm64"
#
# Environment Variables:
#   Required:
#     - QUAY_RHACS_ENG_RW_USERNAME, QUAY_RHACS_ENG_RW_PASSWORD
#     - QUAY_STACKROX_IO_RW_USERNAME, QUAY_STACKROX_IO_RW_PASSWORD
#   Optional:
#     - ROX_OPERATOR_SKIP_PROTO_GENERATED_SRCS (default: true)

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

usage() {
    echo "Build and push operator images for all brandings and architectures"
    echo ""
    echo "Usage: $0 <github_event_name> <github_ref_name> <archs>"
    echo ""
    echo "Arguments:"
    echo "  github_event_name   GitHub event (e.g., 'push', 'pull_request')"
    echo "  github_ref_name     GitHub ref (e.g., 'master', 'feature-branch')"
    echo "  archs               Space-separated list of architectures (e.g., 'amd64 arm64')"
    echo ""
    echo "Example:"
    echo "  $0 pull_request main 'amd64 arm64'"
}

build_operator_image() {
    local branding="$1"
    local arch="$2"

    github_group "Building operator image: ${branding} ${arch}"

    info "Extracting binaries for ${arch}"
    tar xzf "binaries/${arch}/go-binaries-build.tgz"

    info "Building operator image"
    GOARCH="${arch}" retry 6 true make -C operator/ docker-build

    github_endgroup
}

push_operator_image_for_arch() {
    local push_context="$1"
    local branding="$2"
    local arch="$3"

    github_group "Pushing operator image: ${branding} ${arch}"

    info "Pushing architecture-specific image"
    if ! push_operator_image "${push_context}" "${branding}" "${arch}"; then
        die "push_operator_image failed for ${branding} ${arch}"
    fi

    info "Successfully pushed operator image for ${branding} ${arch}"
    github_endgroup
}

build_and_push_branding() {
    local push_context="$1"
    local branding="$2"
    shift 2
    local archs=("$@")

    export ROX_PRODUCT_BRANDING="${branding}"

    # Set registry based on branding
    local quay_org
    if [[ "${branding}" == "RHACS_BRANDING" ]]; then
        quay_org="rhacs-eng"
    else
        quay_org="stackrox-io"
    fi

    info "Docker login to quay.io/${quay_org}"
    registry_rw_login "quay.io/${quay_org}"

    # Build and push each architecture
    for arch in "${archs[@]}"; do
        local tag
        tag="$(make --quiet --no-print-directory -C operator tag)"
        local image="quay.io/${quay_org}/stackrox-operator:${tag}"

        build_operator_image "${branding}" "${arch}"
        push_operator_image_for_arch "${push_context}" "${branding}" "${arch}"
    done

    local arch_csv
    arch_csv=$(IFS=,; echo "${archs[*]}")

    info "Creating manifest for architectures: ${arch_csv}"
    push_operator_manifest_lists "${push_context}" "${branding}" "${arch_csv}"

    info "Completed builds and manifest for ${branding}"
}

main() {
    if [[ "$#" -ne 3 ]]; then
        usage
        die "Invalid number of arguments"
    fi

    local github_event_name="$1"
    local github_ref_name="$2"
    local archs_string="$3"

    info "Starting operator build and push workflow"
    info "Event: ${github_event_name}, Ref: ${github_ref_name}"
    info "Architectures: ${archs_string}"

    # Source CI library for push functions
    # shellcheck source=../../scripts/ci/lib.sh
    source "${SCRIPTS_ROOT}/scripts/ci/lib.sh"

    # Determine push context
    local push_context=""
    if [[ "${github_event_name}" == "push" ]] && [[ "${github_ref_name}" == "master" ]]; then
        push_context="merge-to-master"
        info "Push context: ${push_context}"
    fi

    # Convert space-separated architectures to array
    read -ra archs <<< "${archs_string}"

    info "Building operator for ${#archs[@]} architecture(s): ${archs[*]}"

    # Build for each branding
    for branding in RHACS_BRANDING STACKROX_BRANDING; do
        github_group "Building for branding: ${branding}"
        build_and_push_branding "${push_context}" "${branding}" "${archs[@]}"
        github_endgroup
    done

    info "All operator images and manifests built and pushed successfully"
}

main "$@"
