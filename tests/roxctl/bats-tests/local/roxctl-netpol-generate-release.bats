#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
    command -v yq >/dev/null || skip "Tests in this file require yq"
    echo "Using yq version: '$(yq4.16 --version)'" >&3
    # as of Aug 2022, we run yq version 4.16.2
    # remove binaries from the previous runs
    rm -f "$(roxctl-development-cmd)" "$(roxctl-release-cmd)"
    echo "Testing roxctl version: '$(roxctl-release version)'" >&3
}

setup() {
  out_dir="$(mktemp -d -u)"
  ofile="$(mktemp)"
}

teardown() {
  rm -rf "$out_dir"
  rm -f "$ofile"
}

@test "roxctl-release generate netpol should not respect ROX_ROXCTL_NETPOL_GENERATE feature-flag at runtime" {
  export ROX_ROXCTL_NETPOL_GENERATE='true'
  run roxctl-release generate netpol "$out_dir"
  assert_failure
  assert_line --partial 'unknown command "generate"'
}


