#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

push_images() {
    info "Will push images built in CI"

    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: push_images <brand>"
    fi

    info "Images from OpenShift CI builds:"
    env | grep IMAGE || true

    [[ "${OPENSHIFT_CI:-false}" == "true" ]] || { die "Only supported in OpenShift CI"; }

    local brand="$1"
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

    push_main_image_set "$push_context" "$brand"

    if is_in_PR_context && [[ "$brand" == "STACKROX_BRANDING" ]]; then
        comment_on_pr
    fi
}

comment_on_pr() {
    info "Adding a comment with the build tag to the PR"

    # TODO(RS-509) - remove this when hub-comment is added to rox-ci-image
    if ! command -v "hub-comment" >/dev/null 2>&1; then
        wget --quiet https://github.com/joshdk/hub-comment/releases/download/0.1.0-rc6/hub-comment_linux_amd64
        chmod +x ./hub-comment_linux_amd64
        hub_comment() {
            ./hub-comment_linux_amd64 "$@"
        }
    else
        hub_comment() {
            hub-comment "$@"
        }
    fi

    # hub-comment is tied to Circle CI env
    local url
    url=$(get_pr_details | jq -r '.html_url')
    export CIRCLE_PULL_REQUEST="$url"

    local sha
    sha=$(get_pr_details | jq -r '.head.sha')
    sha=${sha:0:7}
    export _SHA="$sha"

    local tag
    tag=$(make tag)
    export _TAG="$tag"

    local tmpfile
    tmpfile=$(mktemp)
    cat > "$tmpfile" <<- EOT
Images are ready for the commit at {{.Env._SHA}}.

To use with deploy scripts, first \`export MAIN_IMAGE_TAG={{.Env._TAG}}\`.
EOT

    hub_comment -type build -template-file "$tmpfile"
}

push_images "$@"
