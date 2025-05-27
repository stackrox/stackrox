#!/usr/bin/env bash

# This script determines where quay.io/rhacs-eng/* images should be built and pushed and communicates that via stdout.
#
# For non-release branches, PRs and tags we want to build both:
# 1. quay.io/rhacs-eng/<image>:<tag> in GHA, and
# 2. quay.io/rhacs-eng/<image>:<tag>-fast in Konflux.
#
# For releases and RCs we need GHA builds to be suppressed and Konflux builds go to quay.io/rhacs-eng/<image>:<tag>
# (without "-fast" suffix) because this way image tag information which is "baked" in the Konflux-built binaries is
# correct for self-managed (on-prem) release (a.k.a. Stable Stream) and we can further release these images to
# customers.
#
# We want the same for PRs targeting the release branches because we want E2E tests run against Konflux-built images
# just as it happens for release branch pushes.
#
# Additionally, the same behavior will be in any PR when the PR source branch has 'konflux-release-like' in its name.
#
# Note: this script is also called by https://github.com/stackrox/konflux-tasks/blob/main/tasks/determine-image-tag-task.yaml

set -euo pipefail

# Log to stderr to not mess up stdout of any calling code.
function log() {
    >&2 echo "$@"
}

script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
holdfile="${script_dir}/should-konflux-replace-gha-build.hold"

if [[ ( -z "${SOURCE_BRANCH:-}" || -z "${TARGET_BRANCH:-}" ) && -z "${GITHUB_REF:-}" ]]; then
    log "Either SOURCE_BRANCH+TARGET_BRANCH or GITHUB_REF must be set"
    exit 2
fi

# Branch or tag name when in Konflux CI.
# '<branch_name>' for branch push, 'refs/tags/<tag_name>' for tag push.
# For PRs this is '<branch_name>' where the PR is targeted to be merged.
log "Konflux TARGET_BRANCH: ${TARGET_BRANCH:-}"

# Same value as $TARGET_BRANCH for tag and branch pushes. For PRs this is '<branch_name>' of the PR source branch.
log "Konflux SOURCE_BRANCH: ${SOURCE_BRANCH:-}"

# Note that $TARGET_BRANCH and $SOURCE_BRANCH must be explicitly made available as environment variables by/in the
# Tekton step, i.e. they are not provided out of the box unlike the GITHUB_* variables in GHA.

# Branch or tag name when in GHA CI.
# 'refs/heads/<branch_name>' for branch push, 'refs/tags/<tag_name>' for tag push, 'refs/pull/<pr_number>/merge' for PR.
log "GitHub GITHUB_REF: ${GITHUB_REF:-}"

# '<branch_name>' of the PR target branch when it's in GHA CI.
log "GitHub GITHUB_BASE_REF: ${GITHUB_BASE_REF:-}"

# '<branch_name>' of the PR source branch when it's in GHA CI.
log "GitHub GITHUB_HEAD_REF: ${GITHUB_HEAD_REF:-}"

the_ref="${TARGET_BRANCH:-${GITHUB_REF}}"
pr_source_branch="${SOURCE_BRANCH:-}"

if [[ "${GITHUB_REF:-}" == refs/pull/*/merge ]]; then
    if [[ -z "${GITHUB_BASE_REF:-}" || -z "${GITHUB_HEAD_REF:-}" ]]; then
        log "Both GITHUB_BASE_REF and GITHUB_HEAD_REF must be set for PRs"
        exit 3
    fi

    the_ref="${GITHUB_BASE_REF}"
    pr_source_branch="${GITHUB_HEAD_REF}"
fi

if [[ "${pr_source_branch}" == *konflux-release-like* ]]; then
    log "This looks like a PR branch containing the magic string. GHA quay.io/rhacs-eng/* builds must be suppressed in favor of the Konflux ones."
    echo "BUILD_AND_PUSH_ONLY_KONFLUX"
    exit
fi

if grep -qE '^((refs/heads/)?release-[0-9a-z]+\.[0-9a-z]+|refs/tags/[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?)$' <<< "${the_ref}"; then
    log "This looks like a release branch or tag push. GHA quay.io/rhacs-eng/* builds must be suppressed in favor of the Konflux ones."
    if [[ -f "${holdfile}" ]]; then
        # TODO(ROX-29357): remove the holdfile logic after our tests are happy with Konflux-built product.
        log "... would have done that but the 'holdfile' ${holdfile} exists and so not suppressing GHA."
        echo "BUILD_AND_PUSH_BOTH"
    else
        echo "BUILD_AND_PUSH_ONLY_KONFLUX"
    fi
else
    log "This does not look like a release branch or tag push. Both GHA and Konflux should build and push quay.io/rhacs-eng/* images (with different tags)."
    echo "BUILD_AND_PUSH_BOTH"
fi
