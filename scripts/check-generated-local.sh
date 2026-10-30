#!/usr/bin/env bash
# Runs CI's generated-code checks (scripts/ci/jobs/check-generated.sh) against HEAD
# in a reusable scratch git worktree. The CI script resets, deletes and rewrites files,
# so it must never run in the developer's working tree.
#
# Override the scratch location with CHECK_GENERATED_WORKTREE.

set -euo pipefail

repo_root="$(cd "$(git rev-parse --show-toplevel)" && pwd -P)"
sha="$(git -C "$repo_root" rev-parse HEAD)"
short_sha="$(git -C "$repo_root" rev-parse --short HEAD)"

if [[ "$(git -C "$repo_root" rev-parse --is-shallow-repository)" != "false" ]]; then
    echo >&2 "check-generated-local: this clone is shallow, but code generation needs full history and release tags."
    echo >&2 "Run 'git fetch --unshallow --tags' first."
    exit 2
fi

if [[ -n "$(git -C "$repo_root" status --porcelain)" ]]; then
    echo "check-generated-local: note: uncommitted changes are not included, checking last commit ${short_sha} only"
fi

dir="${CHECK_GENERATED_WORKTREE:-${XDG_CACHE_HOME:-$HOME/.cache}/stackrox-check-generated/wt}"
created_dir=""
if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    created_dir="$dir"
fi
dir="$(cd "$dir" && pwd -P)"

if [[ "$dir" == "$repo_root" || "$dir" == "$repo_root"/* ]]; then
    [[ -z "$created_dir" ]] || rmdir "$created_dir"
    echo >&2 "check-generated-local: scratch worktree ${dir} must not be the repository or inside it"
    exit 2
fi

if git -C "$repo_root" worktree list --porcelain | grep -Fxq "worktree ${dir}"; then
    # Only reuse a worktree this script created, never one of the developer's own.
    if [[ ! -e "$(git -C "$dir" rev-parse --absolute-git-dir)/check-generated-local" ]]; then
        echo >&2 "check-generated-local: ${dir} is a worktree not created by this script, refusing to reset it"
        exit 2
    fi
    git -C "$dir" checkout --detach -f "$sha"
else
    git -C "$repo_root" worktree prune
    git -C "$repo_root" worktree add --detach "$dir" "$sha"
    touch "$(git -C "$dir" rev-parse --absolute-git-dir)/check-generated-local"
fi
# Keep downloaded tools (root, operator/ and config-controller/ caches) between runs.
git -C "$dir" clean -ffdx -e .gotools -e .proto

fail_flag_dir="$(mktemp -d)"
trap 'rm -rf "$fail_flag_dir"' EXIT
FAIL_FLAG="${fail_flag_dir}/fail"

rc=0
# Match the CI job's environment (.github/workflows/style.yaml). CI=true makes `make deps`
# skip `go mod tidy`, and the branding decides the operator bundle names.
(cd "$dir" && CI=true ROX_PRODUCT_BRANDING=RHACS_BRANDING FAIL_FLAG="$FAIL_FLAG" \
    scripts/ci/jobs/check-generated.sh) || rc=$?

if [[ "$rc" -eq 0 ]]; then
    echo "check-generated-local: PASS for ${short_sha}"
else
    echo "check-generated-local: FAIL for ${short_sha} (scratch worktree: ${dir})"
fi
exit "$rc"
