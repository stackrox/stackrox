#!/usr/bin/env bats

load "../helpers.bash"
out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
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

@test "roxctl-development connectivity-map should show deprecation info" {
  run roxctl-development connectivity-map
  assert_failure
  assert_line --partial "is deprecated"
}
