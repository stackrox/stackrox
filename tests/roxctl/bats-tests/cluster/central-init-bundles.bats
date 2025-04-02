#!/usr/bin/env bats

load "../helpers.bash"

output=""

setup_file() {
  local -r roxctl_version="$(roxctl-development version || true)"
  echo "Testing roxctl version: '${roxctl_version}'" >&3

  [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
  [[ -n "${ROX_ADMIN_PASSWORD}" ]] || fail "Environment variable 'ROX_ADMIN_PASSWORD' required"
}

setup() {
  output="$(mktemp -d -u)"
}

teardown() {
  rm -rf "${output}"
}

@test "roxctl central init-bundles fetch-ca" {
  run roxctl_authenticated central init-bundles fetch-ca --output ${output}
  assert_success
  assert_file_exist "${output}"
}
