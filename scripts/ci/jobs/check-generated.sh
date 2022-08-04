#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

go mod tidy

# shellcheck disable=SC2016
echo 'Ensure that generated files are up to date. (If this fails, run `make proto-generated-srcs && make go-generated-srcs` and commit the result.)'
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
            cp /tmp/untracked-new "${ARTIFACTS_DIR}/untracked-new"
        fi

        exit 1
    fi
}
generated_files-are-up-to-date

# shellcheck disable=SC2016
echo 'Check operator files are up to date (If this fails, run `make -C operator manifests generate bundle` and commit the result.)'
function check-operator-generated-files-up-to-date() {
    echo 'Checking consistency between EXPECTED_GO_VERSION file and Go version in operator Dockerfile'
    make -C operator/ check-expected-go-version
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
