#!/usr/bin/env bats

load "../helpers.bash"
out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
    command -v yq >/dev/null || skip "Tests in this file require yq"
    echo "Using yq version: '$(yq4.16 --version)'" >&3
    # as of Aug 2022, we run yq version 4.16.2
    # remove binaries from the previous runs
    delete-outdated-binaries "$(roxctl-development version)"
    echo "Testing roxctl version: '$(roxctl-development version)'" >&3
}

setup() {
  out_dir="$(mktemp -d -u)"
  ofile="$(mktemp)"
}

teardown() {
  rm -rf "$out_dir"
  rm -f "$ofile"
}

@test "roxctl-development generate netpol should show deprecation info" {
  run roxctl-development generate netpol
  assert_failure
  assert_line --partial "is deprecated"
}
