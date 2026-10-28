#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export CI_SELECT_ROOT="$root"
# A failed rebuild must not execute a stale binary from an earlier build.
if [[ "${CI_SELECT_BUILD_OUTCOME:-success}" == success ]] && "$root/ci/select/bin/ci-select" gha "$@"; then
    exit 0
fi
echo "CI selector build or execution failed; the job runs" >&2
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    if [[ $# -gt 0 ]]; then
        printf 'action=run\n' >>"$GITHUB_OUTPUT"
    else
        printf 'decisions={}\n' >>"$GITHUB_OUTPUT"
    fi
fi
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    printf 'CI selector build or execution failed; the job runs.\n' >>"$GITHUB_STEP_SUMMARY"
fi
