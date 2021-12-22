#! /usr/bin/env bash

set -uo pipefail

BATS_TESTS="${1:-tests/roxctl/bats-tests}"
CIRCLECI="${CIRCLECI:-}"
echo "Using Bats version: $(bats --version)"
echo "Testing roxctl version: '$(roxctl version)'"

# All flags but --tap require at least Bats v1.5.0
BATS_FLAGS=( "--print-output-on-failure" "--verbose-run" "--show-output-of-passing-tests" )
if [[ $CIRCLECI == "true" ]]; then
  mkdir -p roxctl-test-output
  BATS_FLAGS+=( "--report-formatter" "junit" "--output" "roxctl-test-output" )
fi

# Running all bats test suites found in the directory
echo "Running Bats with parameters: " "${BATS_FLAGS[@]}"
bats "${BATS_FLAGS[@]}" "${BATS_TESTS}"
