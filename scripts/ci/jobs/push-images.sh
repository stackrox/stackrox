#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

push_images() {
    info "Will push images built in CI"

    env | grep IMAGE

    [[ "${OPENSHIFT_CI:-false}" == "true" ]] || { die "Only supported in OpenShift CI"; }

    local brand="$1"
    local branch
    branch=$(get_pr_details | jq -r '.head.ref')
    if [[ "$branch" == "null" ]]; then
        branch="master"
    fi

    oc registry login

    push_main_image_set "$branch" "$brand"
}

push_images "STACKROX_BRANDING"
