#!/usr/bin/env bash
# EXPERIMENT (do not merge): parallel variants of check-generated.sh.
# Usage: check-generated-experiment.sh <tidy|generated|operator|misc|all|parallel>
#
# Groups mirror the checks in check-generated.sh, which runs them serially.
# "parallel" runs the four groups concurrently in separate git worktrees
# (the diff checks compare the whole tree, so groups must not share a checkout).
# Failure reporting is simplified relative to the production script: GitHub
# annotations and exit codes are preserved, junit reporting is not.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail
export SHELLOPTS

GROUP="${1:?usage: $0 <tidy|generated|operator|misc|all|parallel>}"

EXPERIMENT_STATE_DIR=$(mktemp -d)
export EXPERIMENT_STATE_DIR ROOT
trap 'rm -rf "$EXPERIMENT_STATE_DIR"' EXIT
FAIL_FLAG="$EXPERIMENT_STATE_DIR/fail"

source "$ROOT/scripts/ci/generated-check-output.sh"

mark_failed() {
    echo "$1" >> "$FAIL_FLAG"
    echo "::error::experiment group failed: $1"
}

# ---------------------------------------------------------------- group: tidy
function check-go-mod-tidy() {
    for gomod in $(git ls-files '*/go.mod' 'go.mod'); do
        (cd "$(dirname "$gomod")" && go mod tidy) || exit 1
    done
    echo 'Checking for diffs after go mod tidy...'
    generated_diff
}
export -f check-go-mod-tidy

# ---------------------------------------------------------- group: generated
function generated_files-are-up-to-date() {
    git ls-files --others --exclude-standard >"$EXPERIMENT_STATE_DIR/untracked"

    make proto-generated-srcs

    git grep --files-with-matches "Package mocks is a generated GoMock package." -- '*.go' | xargs rm

    make go-generated-srcs

    local changed=0
    generated_diff || changed=1
    { git ls-files --others --exclude-standard ; cat "${EXPERIMENT_STATE_DIR}/untracked" ; } | sort | uniq -u >"${EXPERIMENT_STATE_DIR}/untracked-new"

    if [[ -s "${EXPERIMENT_STATE_DIR}/untracked-new" ]]; then
        echo 'ERROR: Found new untracked files after generation.'
        cat "${EXPERIMENT_STATE_DIR}/untracked-new"
        return 1
    fi
    return "$changed"
}
export -f generated_files-are-up-to-date

# ---------------------------------------------------------- group: operator
function check-operator-generated-files-up-to-date() {
    make -C operator/ generate
    make -C operator/ manifests
    echo 'Checking for diffs after making generate and manifests...'
    generated_diff

    make -C operator/ bundle
    echo 'Checking for diffs after making bundle...'
    generated_diff

    make -C operator/ chart
    echo 'Expanding the operator helm chart...'
    helm template --namespace rhacs-operator-system rhacs-operator ./operator/dist/chart/ > operator/dist/chart.yaml
    echo 'Downloading yq...'
    make -C operator/ yq
    yq=$(make --no-print-directory --silent -C operator/ which-yq)
    echo 'Normalizing the manifests...'
    $yq -P ea '[.] | sort_by(.kind, .metadata.name)
      | del(.[].metadata.annotations.["helm.sh/resource-policy"])
      | del(.[] | select(.kind == "Deployment").spec.template.spec.imagePullSecrets)
      | .[] | splitDoc | ... comments=""' \
      operator/dist/chart.yaml > operator/dist/chart-sorted.yaml
    $yq -P ea '[.] | sort_by(.kind, .metadata.name)
      | filter(.kind != "Namespace")
      | del(.[] | select(.kind == "Deployment").spec.template.spec.imagePullSecrets)
      | .[] | splitDoc | ... comments=""' \
      operator/dist/install.yaml > operator/dist/install-sorted.yaml

    echo 'Checking for differences between normalized operator manifest and normalized and expanded operator helm chart...'
    diff -U 10 operator/dist/install-sorted.yaml operator/dist/chart-sorted.yaml
}
export -f check-operator-generated-files-up-to-date

# ------------------------------------------------------------- group: misc
function check-config-controller-generated-files-up-to-date() {
    make config-controller-gen
    echo 'Checking for diffs after making config-controller-gen...'
    generated_diff
}
export -f check-config-controller-generated-files-up-to-date

function check-containerignore-is-in-sync() {
    diff \
        --unified \
        --ignore-blank-lines \
        <(grep -v -e '^#' .containerignore) \
        <(grep -vF -e '/.git/' -e '/image/' -e '/qa-tests-backend/' .dockerignore)
}
export -f check-containerignore-is-in-sync

function check-shellcheck-failing-list() {
    make update-shellcheck-skip
    echo 'Checking for diffs after updating shellcheck failing list...'
    git diff --exit-code HEAD
}
export -f check-shellcheck-failing-list

run_group() {
    local group="$1" dir="${2:-.}"
    local start=$SECONDS
    echo "::group::experiment group: $group (dir: $dir)"
    local rc=0
    (cd "$dir" && bash -c "set -euo pipefail; export SHELLOPTS; \
        ROOT='$ROOT'; source '$ROOT/scripts/ci/lib.sh'; \
        CHECK_GENERATED_STATE_DIR=\$(mktemp -d); export CHECK_GENERATED_STATE_DIR; \
        source '$ROOT/scripts/ci/generated-check-output.sh'; \
        case '$group' in
            tidy) run_generator_command bash -c check-go-mod-tidy ;;
            generated) run_generator_command bash -c generated_files-are-up-to-date ;;
            operator) run_generator_command bash -c check-operator-generated-files-up-to-date ;;
            misc)
                run_generator_command bash -c check-config-controller-generated-files-up-to-date
                run_generator_command bash -c check-containerignore-is-in-sync
                run_generator_command bash -c check-shellcheck-failing-list
                ;;
        esac") || rc=$?
    echo "::endgroup::"
    local took=$((SECONDS - start))
    echo "TIMING group=$group seconds=$took rc=$rc"
    if (( rc != 0 )); then
        mark_failed "$group (rc=$rc, ${took}s)"
        (cd "$dir" && git reset --hard HEAD >/dev/null 2>&1 || true)
    fi
}

case "$GROUP" in
    tidy|generated|operator|misc)
        run_group "$GROUP"
        ;;
    all)
        run_group tidy
        run_group generated
        run_group operator
        run_group misc
        ;;
    parallel)
        # Each group runs in its own git worktree: the diff-based checks compare
        # the whole working tree, so concurrent groups must not share a checkout.
        WT_BASE=$(mktemp -d)
        for g in generated operator misc; do
            git worktree add --quiet "$WT_BASE/$g" HEAD
        done
        pids=()
        run_group generated "$WT_BASE/generated" & pids+=($!)
        run_group operator "$WT_BASE/operator" & pids+=($!)
        run_group misc "$WT_BASE/misc" & pids+=($!)
        run_group tidy & pids+=($!)
        failed=0
        for p in "${pids[@]}"; do
            wait "$p" || failed=1
        done
        for g in generated operator misc; do
            git worktree remove --force "$WT_BASE/$g" >/dev/null 2>&1 || true
        done
        (( failed == 0 )) || exit 1
        ;;
    *)
        echo "unknown group: $GROUP" >&2
        exit 2
        ;;
esac

if [[ -e "$FAIL_FLAG" ]]; then
    echo "ERROR: Some generated file checks failed:"
    cat "$FAIL_FLAG"
    exit 1
fi
