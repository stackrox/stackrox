#! /usr/bin/env bash

set -euo pipefail

TESTS_OUTPUT="${1:-roxctl-test-output}"
BATS_TESTS="${2:-tests/roxctl/bats-tests}"
echo "Using Bats version: $(bats --version)"
echo "Testing roxctl version: '$(roxctl version)'"

# Requires Bats v1.5.0
BATS_FLAGS=( "--print-output-on-failure" "--verbose-run" )
if [[ -n "${CI:-}" ]]; then
  mkdir -p "$TESTS_OUTPUT"
  BATS_FLAGS+=( "--report-formatter" "junit" "--output" "$TESTS_OUTPUT" )
else
  # Using this causes junit to treat all passing tests with output as failures
  BATS_FLAGS+=( "--show-output-of-passing-tests" )
fi

# Running all bats test suites found in the directory
echo "Running Bats with parameters: " "${BATS_FLAGS[@]}"
bats "${BATS_FLAGS[@]}" "${BATS_TESTS}"
