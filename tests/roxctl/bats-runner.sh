#! /usr/bin/env bash

set -euo pipefail

if (( $# != 2 )); then
  echo >&2 "Error: bats-runner.sh requires 2 arguments: bats-runner.sh <test_output> <tests>"
  exit 1
fi

TESTS_OUTPUT="${1:-roxctl-test-output}"
BATS_TESTS="${2:-tests/roxctl/bats-tests}"
echo "Using Bats version: '$(bats --version)'"
echo "Testing roxctl version: '$(roxctl version)'"

# Requires Bats v1.5.0
BATS_FLAGS=( "--print-output-on-failure" "--verbose-run" "--recursive" )
if [[ -n "${CI:-}" ]]; then
  mkdir -p "$TESTS_OUTPUT"
  BATS_FLAGS+=( "--report-formatter" "junit" "--output" "$TESTS_OUTPUT" )
else
  # Using this causes junit to treat all passing tests with output as failures
  BATS_FLAGS+=( "--show-output-of-passing-tests" )
fi

# Running all bats test suites found in the directory
echo "BATS_FLAGS   : ${BATS_FLAGS[@]}"
echo "BATS_TESTS   : ${BATS_TESTS}"
echo "TESTS_OUTPUT : $TESTS_OUTPUT"
echo "BATS_TESTS   : $BATS_TESTS"

set -x
bats "${BATS_FLAGS[@]}" "${BATS_TESTS}"
