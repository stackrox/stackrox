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

@test "roxctl-release roxctl help should have no duplicated (default false) " {
  run roxctl-release central --help
  assert_line --regexp '^[[:space:]]+--insecure .* to true \(default false\)$'
}

@test "roxctl-release roxctl central whoami help should have no duplicated (default false) " {
  run roxctl-release central whoami --help
  assert_line --regexp '^[[:space:]]+--plaintext .* to true \(default false\)$'
}

@test "roxctl-release roxctl declarative-config create notifier generic help shouldn have no duplicated (default false) " {
  run roxctl-release declarative-config create notifier generic --help
  assert_line --regexp '^[[:space:]]+--webhook-skip-tls-verify .* verification \(default false\)$'
}
