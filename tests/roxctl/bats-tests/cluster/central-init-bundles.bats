#!/usr/bin/env bats

load "../helpers.bash"

output_dir=""

setup_file() {
  local -r roxctl_version="$(roxctl-development version || true)"
  echo "Testing roxctl version: '${roxctl_version}'" >&3

  [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
  [[ -n "${ROX_ADMIN_PASSWORD}" ]] || fail "Environment variable 'ROX_ADMIN_PASSWORD' required"
}

setup() {
  output_dir="$(mktemp -d)"
}

teardown() {
  rm -rf "${output_dir}"
}

@test "roxctl central init-bundles fetch-ca" {
  run roxctl_authenticated central init-bundles fetch-ca --output ${output_dir}/ca-config.yaml
  assert_success
  assert_file_exist "${output_dir}/ca-config.yaml"
}
