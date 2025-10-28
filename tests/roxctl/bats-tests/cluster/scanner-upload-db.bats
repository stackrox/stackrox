#!/usr/bin/env bats

load "../helpers.bash"

temp_dir=""

setup_file() {
  local -r roxctl_version="$(roxctl-development version || true)"
  echo "Testing roxctl version: '${roxctl_version}'" >&3

  command -v curl || skip "Command 'curl' required."
  [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
  [[ -n "${ROX_ADMIN_PASSWORD}" ]] || fail "Environment variable 'ROX_ADMIN_PASSWORD' required"
}

setup() {
  temp_dir="$(mktemp -d)"
}

teardown() {
  rm -rf "${temp_dir}"
}

@test "[no-option] roxctl scanner upload-db" {
  run roxctl_authenticated scanner upload-db
  assert_failure
  assert_output --partial '"scanner-db-file" not set'
}

@test "[non-zip] roxctl scanner upload-db" {
  echo 'Just text' > "${temp_dir}/test-invalid-scanner-vuln-updates.zip"

  run roxctl_authenticated scanner upload-db --scanner-db-file "${temp_dir}/test-invalid-scanner-vuln-updates.zip"
  assert_failure
  assert_output --partial 'not a valid zip file'
}

# TODO(ROX-29096): Make this test pass with Scanner V4 enabled.
@test "[zip] roxctl scanner upload-db" {
  if [[ "${ROX_SCANNER_V4}" == "true" ]]; then
      skip "https://issues.redhat.com/browse/ROX-28949"
  fi

  run roxctl_authenticated scanner upload-db --scanner-db-file "${temp_dir}/test-scanner-vuln-updates.zip"
  assert_success
}
