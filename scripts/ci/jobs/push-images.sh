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

    if [[ "$brand" == "STACKROX_BRANDING" ]] && [[ -n "${MAIN_IMAGE}" ]]; then
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

    if [[ -n "${MAIN_IMAGE}" ]]; then
        if is_OPENSHIFT_CI && is_in_PR_context && pr_has_label "turbo-build" && [[ "$brand" == "RHACS_BRANDING" ]]; then
            info "Images were built and pushed elsewhere, skipping it here."
        else
            push_main_image_set "$push_context" "$brand"
            push_matching_collector_scanner_images "$brand"
        fi
    fi
    if [[ -n "${PIPELINE_DOCS_IMAGE:-}" ]]; then
        push_docs_image
    fi
    if [[ -n "${MAIN_RCD_IMAGE:-}" ]]; then
        push_race_condition_debug_image
    fi
    if [[ -n "${OPERATOR_IMAGE:-}" ]]; then
        if is_OPENSHIFT_CI && is_in_PR_context && pr_has_label "turbo-build"; then
            info "Operator images were built and pushed elsewhere, skipping it here."
        else
            push_operator_image_set "$push_context" "$brand"
        fi
    fi

    if is_in_PR_context && [[ "$brand" == "STACKROX_BRANDING" ]] && [[ -n "${MAIN_IMAGE}" ]]; then
        add_build_comment_to_pr || {
            info "Could not add a comment to the PR"
        }
    fi
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
        build_url="https://prow.ci.openshift.org/?repo=stackrox%2Fstackrox&job=*stackrox*night*"
        # send to #nightly-ci-runs
        webhook_url="${NIGHTLY_WORKFLOW_NOTIFY_WEBHOOK}"
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
