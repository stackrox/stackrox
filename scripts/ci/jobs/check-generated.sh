#!/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

go mod tidy


# shellcheck disable=SC2016
echo 'Ensure that generated files are up to date. (If this fails, run `make proto-generated-srcs && make go-generated-srcs` and commit the result.)'
function generated_files-are-up-to-date() {

    ln -s /go/src/github.com/stackrox /go/src/github.com/rox

    git ls-files --others --exclude-standard >/tmp/untracked
    make proto-generated-srcs
    # Print the timestamp along with each new line of output, so we can track how long each command takes
    make go-generated-srcs 2>&1 | while IFS= read -r line; do printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$line"; done
    git diff --exit-code HEAD
    { git ls-files --others --exclude-standard ; cat /tmp/untracked ; } | sort | uniq -u >/tmp/untracked-new

    store_test_results /tmp/untracked-new untracked-new

    if [[ -s /tmp/untracked-new ]]; then
        # shellcheck disable=SC2016
        echo 'Found new untracked files after running `make proto-generated-srcs` and `make go-generated-srcs`. Did you forget to `git add` generated mocks and protos?'
        cat /tmp/untracked-new
        exit 1
    fi
}
generated_files-are-up-to-date

echo 'Ensure that all TODO references to fixed tickets are gone'
if is_CIRCLECI; then
    "$SCRIPTS_ROOT/.circleci/check-pr-fixes.sh"
else
    "$SCRIPTS_ROOT/.openshift-ci/check-pr-fixes.sh"
fi

echo 'Ensure that there are no TODO references that the developer has marked as blocking a merge'
echo "Matches comments of the form TODO(x), where x can be \"DO NOT MERGE/don't-merge\"/\"dont-merge\"/similar"
./scripts/check-todos.sh 'do\s?n.*merge'

# shellcheck disable=SC2016
echo 'Check operator files are up to date (If this fails, run `make -C operator manifests generate bundle` and commit the result.)'
function check-operator-generated-files-up-to-date() {
    set -e
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
check-operator-generated-files-up-to-date
