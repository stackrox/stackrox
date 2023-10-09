#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

go mod tidy

FAIL_FLAG="/tmp/fail"

# shellcheck disable=SC2016
info 'Ensure that generated files are up to date. (If this fails, run `make proto-generated-srcs && make go-generated-srcs` and commit the result.)'
function generated_files-are-up-to-date() {
    git ls-files --others --exclude-standard >/tmp/untracked
    make proto-generated-srcs
    # Print the timestamp along with each new line of output, so we can track how long each command takes
    make go-generated-srcs 2>&1 | while IFS= read -r line; do printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$line"; done
    git diff --exit-code HEAD
    { git ls-files --others --exclude-standard ; cat /tmp/untracked ; } | sort | uniq -u >/tmp/untracked-new

    if [[ -s /tmp/untracked-new ]]; then
        # shellcheck disable=SC2016
        echo 'ERROR: Found new untracked files after running `make proto-generated-srcs` and `make go-generated-srcs`. Did you forget to `git add` generated mocks and protos?'
        cat /tmp/untracked-new

        if is_OPENSHIFT_CI; then
            cp /tmp/untracked-new "${ARTIFACT_DIR:-}/untracked-new"
        fi
        return 1
    fi
}
generated_files-are-up-to-date || {
    save_junit_failure "Check_Generated_Files" \
        "Found new untracked files after running \`make proto-generated-srcs\` and \`make go-generated-srcs\`" \
        "$(cat /tmp/untracked-new)"
    git reset --hard HEAD
    echo generated_files-are-up-to-date >> "$FAIL_FLAG"
}

# shellcheck disable=SC2016
info 'Check operator files are up to date (If this fails, run `make -C operator manifests generate bundle` and commit the result.)'
function check-operator-generated-files-up-to-date() {
    make -C operator/ generate
    make -C operator/ manifests
    echo 'Checking for diffs after making generate and manifests...'
    git diff --exit-code HEAD
    make -C operator/ bundle
    echo 'Checking for diffs after making bundle...'
    echo 'If this fails, check if the invocation of the normalize-metadata.py script in operator/Makefile'
    echo 'needs to change due to formatting changes in the generated files.'
    git diff --exit-code HEAD
}
check-operator-generated-files-up-to-date || {
    save_junit_failure "Check_Operator_Generated_Files" \
        "Operator generated files are not up to date" \
        "$(git diff HEAD || true)"
    git reset --hard HEAD
    echo check-operator-generated-files-up-to-date >> "$FAIL_FLAG"
}

info 'Check .{docker,container}ignore files are up to date (If this fails, follow instructions in .containerignore to update the file.)'
function check-container-ignore-files-up-to-date() {
    diff -u -I '/.git/|#.*' .containerignore .dockerignore
}
check-container-ignore-files-up-to-date || {
    save_junit_failure "Check_Container_Ignore_Files" \
        "Container ignore files are not up to date" \
        "$(diff -u -I '/.git/|#.*' .containerignore .dockerignore || true)"
    git reset --hard HEAD
    echo check-container-ignore-files-up-to-date >> "$FAIL_FLAG"
}

# shellcheck disable=SC2016
echo 'Check if a script that was on the failed shellcheck list is now fixed. (If this fails, run `make update-shellcheck-skip` and commit the result.)'
function check-shellcheck-failing-list() {
    make update-shellcheck-skip
    echo 'Checking for diffs after updating shellcheck failing list...'
    git diff --exit-code HEAD
}
check-shellcheck-failing-list || {
    save_junit_failure "Check_Shellcheck_Skip_List" \
        "Check if a script that is listed in scripts/style/shellcheck_skip.txt is now free from shellcheck errors" \
        "$(git diff HEAD || true)"
    git reset --hard HEAD
    echo check-shellcheck-failing-list >> "$FAIL_FLAG"
}

if [[ -e "$FAIL_FLAG" ]]; then
    echo "ERROR: Some generated file checks failed:"
    cat "$FAIL_FLAG"
    exit 1
fi
