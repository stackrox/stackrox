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

    if [[ "$brand" == "STACKROX_BRANDING" ]]; then
        slack_build_notice "$tag"
    fi

    if is_release_version "$tag"; then
        check_docs "${tag}"
        check_scanner_and_collector_versions
    else
        info "Not checking docs/ & version files for non releases"
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

    push_main_image_set "$push_context" "$brand"
    push_matching_collector_scanner_images "$brand"
    if [[ -n "${PIPELINE_DOCS_IMAGE:-}" ]]; then
        push_docs_image
    fi
    push_race_condition_debug_image

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

slack_build_notice() {
    info "Slack a build notice"

    if [[ "$#" -lt 1 ]]; then
        die "missing arg. usage: slack_build_notice <tag>"
    fi

    local tag="$1"

    [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]] || is_nightly_run || {
        info "Skipping step as this is not a release, RC or nightly build"
        return 0
    }

    local build_url
    local webhook_url
    if [[ "$tag" =~ $RELEASE_RC_TAG_BASH_REGEX ]]; then
        local release
        release="$(get_release_stream "$tag")"
        build_url="https://prow.ci.openshift.org/?repo=stackrox%2Fstackrox&job=*release-$release*"
        if is_release_test_stream "$tag"; then
            # send to #slack-test when testing the release process
            webhook_url="${SLACK_MAIN_WEBHOOK}"
        else
            # send to #eng-release
            webhook_url="${RELEASE_WORKFLOW_NOTIFY_WEBHOOK}"
        fi
    elif is_nightly_run; then
        build_url="https://prow.ci.openshift.org/?repo=stackrox%2Fstackrox&job=periodic*nightly*"
        if is_in_PR_context && pr_has_label "simulate-nightly-run"; then
            # send to #slack-test when testing nightlies
            webhook_url="${SLACK_MAIN_WEBHOOK}"
        else
            # send to #nightly-ci-runs
            webhook_url="${NIGHTLY_WORKFLOW_NOTIFY_WEBHOOK}"
        fi
    else
        die "unexpected"
    fi

    jq -n \
    --arg build_url "$build_url" \
    --arg tag "$tag" \
    '{"text": ":prow: Prow build for tag `\($tag)` started! Check the status of the build under the following URL: \($build_url)"}' \
| curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url"
}

push_images "$@"
