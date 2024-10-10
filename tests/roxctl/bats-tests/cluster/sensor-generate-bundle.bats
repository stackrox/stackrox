#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || fail "API_ENDPOINT environment variable required"
  [[ -n "$ROX_ADMIN_PASSWORD" ]] || fail "ROX_ADMIN_PASSWORD environment variable required"
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

fetch_bundle() {
  local name="$1";shift
  bundle_output="$(mktemp -d -u)"
  run roxctl_authenticated sensor get-bundle "$name" \
    --output-dir "$bundle_output" "$@"
  assert_success
  rm -rf "$bundle_output"
}

run_generate_and_get_bundle_test() {
  local flavor="$1";shift
  local name="$1";shift
  generate_bundle "$flavor" --name "$name" "$@"
  assert_success
  fetch_bundle "$name"
  delete_cluster "$name"
}

@test "[k8s] roxctl sensor generate and get-bundle" {
  run_generate_and_get_bundle_test k8s "k8s-test-cluster"
}

@test "[openshift3] roxctl sensor generate and get-bundle" {
  run_generate_and_get_bundle_test openshift "oc3-test-cluster" --openshift-version 3
}

@test "[openshift4] roxctl sensor generate and get-bundle" {
  run_generate_and_get_bundle_test openshift "oc4-test-cluster" --openshift-version 4
}

@test "roxctl sensor generate fails if bundle name is not provided" {
  generate_bundle k8s
  assert_failure
}
