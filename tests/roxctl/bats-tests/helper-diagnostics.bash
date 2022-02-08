#!/usr/bin/env bash

diagout() {
  if [[ -n "${TESTS_OUTPUT}" ]]; then
    echo "${TESTS_OUTPUT}/diag"
  else
    echo "/dev/null"
  fi
}

# diagecho is bats-echo and it uses descriptor 3 instead of 1
diagecho() {
  local out
  out="$(diagout)"
  # empty line for readability
  echo "" >> "$out/$BATS_TEST_NUMBER-$BATS_TEST_NAME.log"
  echo "${@}" >> "$out/$BATS_TEST_NUMBER-$BATS_TEST_NAME.log"
}

diag_header() {
  local out
  out="$(diagout)"
  diagecho "diagout: '$out'"
  diagecho "=== DIAG test '$BATS_TEST_NAME' ==="

  local out_dir="${1}"
  [[ -n "$out_dir" ]] || { echo "out_dir is not set"; exit 1;}
  if [[ -d "$out_dir" ]]; then
    tree -p -a -h -D -F -o "${out}/tree-${BATS_TEST_NUMBER}-${BATS_TEST_NAME}.log" "$out_dir"
  fi
  # display one level up but only depth 1
  tree -p -a -h -D -F -L 1 -o "${out}/tree-ONE-UP-${BATS_TEST_NUMBER}-${BATS_TEST_NAME}.log" "$out_dir/.."
}

diag_footer() {
  diagecho "=== DIAG end of test '$BATS_TEST_NAME' ===" >&3
}

diagnose() {
  local out_dir="${1}"
  local status="${2:-unknown}"

  diag_header "$out_dir"

  diagecho "run command: '$BATS_RUN_COMMAND'"

  diagecho "run status: '$status'"

  diagecho "BATS_RUN_TMPDIR: '$BATS_RUN_TMPDIR'"
  diagecho "ls -la BATS_RUN_TMPDIR: $(ls -la "$BATS_RUN_TMPDIR")"

  diagecho "BATS_SUITE_TMPDIR: '$BATS_SUITE_TMPDIR'"
  diagecho "ls -la BATS_SUITE_TMPDIR: $(ls -la "$BATS_SUITE_TMPDIR")"

  diagecho "BATS_FILE_TMPDIR: '$BATS_FILE_TMPDIR'"
  diagecho "ls -la BATS_FILE_TMPDIR: $(ls -la "$BATS_FILE_TMPDIR")"

  diagecho "BATS_TEST_TMPDIR: '$BATS_TEST_TMPDIR'"
  diagecho "ls -la BATS_TEST_TMPDIR: $(ls -la "$BATS_TEST_TMPDIR")"

  diagecho "out_dir: '$out_dir'"
  test -d "$out_dir"
  diagecho "test -d '$out_dir': '$?'"
  test -e "$out_dir"
  diagecho "test -e '$out_dir': '$?'"

  diag_footer
}
