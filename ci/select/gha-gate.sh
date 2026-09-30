#!/usr/bin/env bash
# Print the CI decision, or record run/skip for one GitHub Actions job.
# With no argument the plan is printed. With a job name, action=run or
# action=skip is appended to GITHUB_OUTPUT when that file is set.

set -euo pipefail

job="${1:-}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

python="python3"
if ! command -v python3 >/dev/null 2>&1; then
    python="python"
fi

labels_file="$(mktemp)"
files_file="$(mktemp)"
trap 'rm -f "$labels_file" "$files_file"' EXIT

if [[ -n "${PR_LABELS:-}" ]]; then
    printf '%s\n' "${PR_LABELS}" | tr ',' '\n' >"$labels_file"
else
    : >"$labels_file"
fi

files_unknown=0
if [[ -z "${CI_BASE:-}" || -z "${CI_HEAD:-}" ]]; then
    files_unknown=1
elif ! git -C "$root" diff --name-only "${CI_BASE}...${CI_HEAD}" >"$files_file"; then
    echo "CI decision: changed files are unknown; decision-defaults will be used"
    files_unknown=1
fi

cmd=("$python" "$root/ci/select/gha.py")
if [[ -n "$job" ]]; then
    cmd+=(gate --job "$job")
else
    cmd+=(print)
fi
cmd+=(
    --rules "$root/ci/test-domains.toml"
    --defaults "$root/ci/decision-defaults"
    --labels-file "$labels_file"
    --repo "${CI_REPO:-stackrox/stackrox}"
    --pr "${CI_PR:-0}"
    --commit "${CI_COMMIT:-unknown}"
)
if [[ "$files_unknown" -eq 1 ]]; then
    cmd+=(--files-unknown)
else
    cmd+=(--files-file "$files_file")
fi
if [[ "${CI_ENFORCE:-}" == "true" ]]; then
    cmd+=(--enforce)
fi
if [[ -n "${CI_DECISION_OUT:-}" ]]; then
    cmd+=(--decision-out "$CI_DECISION_OUT")
fi

if [[ -z "$job" ]]; then
    "${cmd[@]}" || echo "CI decision could not be printed"
    exit 0
fi

action="run"
if ! action="$("${cmd[@]}")"; then
    echo "CI decision gate failed; ${job} runs"
    action="run"
fi
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf 'action=%s\n' "$action" >>"$GITHUB_OUTPUT"
fi
printf 'CI decision for %s: %s\n' "$job" "$action"
