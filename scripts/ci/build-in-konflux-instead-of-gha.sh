#!/usr/bin/env bash

# For non-release branches, PRs and tags we want to build both:
# 1. quay.io/rhacs-eng/<image>:<tag> in GHA, and
# 2. quay.io/rhacs-eng/<image>:<tag>-fast in Konflux.
#
# For releases and RCs we need GHA builds to be suppressed and Konflux builds go to quay.io/rhacs-eng/<image>:<tag>
# (without "-fast" suffix) because this way image tag information "baked" in the Konflux-built binaries is correct for
# self-managed (on-prem) release (a.k.a. Stable Stream) and we can further release these images to customers.
#
# We want the same for PRs targeting release branches because we want E2E tests run against Konflux-built images just as
# it happens for release branch pushes.
#
# This is what this script determines and communicates via its exit code.
# 0 -> only Konflux without suffix, no GHA.
# No-zero (6) -> both Konflux (with suffix) and GHA.

set -euo pipefail

function log() {
    >&2 echo "$@"
}

if [[ -z "${SOURCE_BRANCH:-}" && -z "${GITHUB_REF:-}" ]]; then
    log "Either SOURCE_BRANCH or GITHUB_REF must be set"
    exit 2
fi

# TODO: support pull requests

# Branch or tag name when in Konflux CI.
# Note that $SOURCE_BRANCH must be manually exposed as the environment variable by/in the Tekton step.
# '<branch_name>' for branch push, 'refs/tags/<tag_name>' for tag push.
log "Konflux SOURCE_BRANCH: ${SOURCE_BRANCH:-}"

# Branch or tag name when in GHA CI.
# 'refs/heads/<branch_name>' for branch push, 'refs/pull/<pr_number>/merge' for PR, 'refs/tags/<tag_name>' for tag push.
log "GitHub GITHUB_REF: ${GITHUB_REF:-}"

the_ref="${SOURCE_BRANCH:-${GITHUB_REF}}"

if grep -qE '^((refs/heads/)?release-[0-9a-z]+\.[0-9a-z]+|refs/tags/[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?)$' <<< "${the_ref}"; then
    log "This looks like a release branch or tag push, GHA quay.io/rhacs-eng/* builds must be suppressed in favor of the Konflux ones."
    exit 0
else
    log "This does not look like a release branch or tag push, both GHA and Konflux should build and push quay.io/rhacs-eng/* images (with different tags)."
    exit 6
fi
