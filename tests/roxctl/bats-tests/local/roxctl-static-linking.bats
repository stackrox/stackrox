#!/usr/bin/env bats

load "../helpers.bash"

setup_file() {
  delete-outdated-binaries "$(roxctl-release version)"
  echo "Testing roxctl version: '$(roxctl-release version)'" >&3
}

@test "roxctl-release linux binary is statically linked" {
  [[ "$(luname)" == "linux" ]] || skip "static linking check only runs on linux"
  local roxctl_bin
  roxctl_bin="$(roxctl-release-cmd)"
  run file -b "$roxctl_bin"
  assert_success
  assert_output --partial "statically linked"
}

@test "roxctl-release linux binary has no dynamic library dependencies" {
  [[ "$(luname)" == "linux" ]] || skip "ldd check only runs on linux"
  local roxctl_bin
  roxctl_bin="$(roxctl-release-cmd)"
  run ldd "$roxctl_bin"
  assert_output --partial "not a dynamic executable"
}
