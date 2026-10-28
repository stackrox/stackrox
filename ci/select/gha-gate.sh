#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export CI_SELECT_ROOT="$root"
if "$root/ci/select/bin/ci-select" gha-ci "$@"; then
    exit 0
fi
echo "CI decision gate failed; the job runs" >&2
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf 'action=run\n' >>"$GITHUB_OUTPUT"
fi
