#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  # remove binaries from the previous runs
  delete-outdated-binaries "$(roxctl-release version)"

  echo "Testing roxctl version: '$(roxctl-release version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
  export out_dir
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-release roxctl help output " {
  diff -u <(roxctl-release --help 2>&1) "tests/roxctl/bats-tests/local/expect/roxctl--help.txt"
}

@test "roxctl-release roxctl central whoami help output " {
  diff -u <(roxctl-release central whoami --help 2>&1) "tests/roxctl/bats-tests/local/expect/roxctl_central_whoami--help.txt"
}

@test "roxctl-release roxctl declarative-config create notifier generic help output " {
  diff -u <(roxctl-release declarative-config create notifier generic --help 2>&1) "tests/roxctl/bats-tests/local/expect/roxctl_declarative-config_create_notifier_generic--help.txt"
}
