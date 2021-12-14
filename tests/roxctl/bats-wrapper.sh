#! /usr/bin/env bash

set -uo pipefail

FAILURES=0
BATS_TESTS_DIR="${1:-tests/roxctl/bats-tests}"

BATS_VERSION="$(bats --version)"
echo "Using Bats version: $BATS_VERSION"
echo "Testing roxctl version: '$(roxctl version)'"

BATS_FLAGS=( "--tap" )
# poor-mans semver compare
if [[ $BATS_VERSION =~ ^Bats\ ([0-9]).([0-9]).([0-9])?$ ]]; then
  if (( "${BASH_REMATCH[1]}" == 1 && "${BASH_REMATCH[2]}" >= 5 )); then
    # Useful flags introduced in v1.5.0
    BATS_FLAGS+=( "--print-output-on-failure" "--verbose-run" "--show-output-of-passing-tests" ) # TODO(PR): consider removing --show-output-of-passing-tests before merging
  fi
fi

# Running all bats test suites found in the directory
echo "Running Bats with parameters: " "${BATS_FLAGS[@]}"
BATS_OUT="$(bats "${BATS_FLAGS[@]}" "${BATS_TESTS_DIR}")"
NUM_FAILED="$(grep -c "^not ok " <<< "$BATS_OUT")"
FAILURES=$((FAILURES + NUM_FAILED))

if (( ! FAILURES )); then
  echo "All tests passed"
  printf "Bats output:\n%s" "$BATS_OUT" # TODO(PR): remove before merging
else
  echo "$FAILURES test failed"
  printf >&2 "Bats output:\n%s" "$BATS_OUT"  # TODO(PR): remove before merging
  exit 1
fi
printf "\nBats testing done.\n"
