#!/usr/bin/env bats

load "helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || skip "API_ENDPOINT environment variable required"
  [[ -n "$ROX_PASSWORD" ]] || skip "ROX_PASSWORD environment variable required"
  export_api_token
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

dev_registry_regex="docker\.io/stackrox"
any_version="[0-9]+\.[0-9]+\."
any_version_latest="[0-9]+\.[0-9]+\.[0-9]+\-latest"
any_version_slim="[0-9]+\.[0-9]+\.[0-9]+\-slim"

@test "roxctl sensor generate: no overrides" {
  name="$(bundle_unique_name)"
  generate_bundle k8s --name "$name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_slim"
  delete_cluster "$name"
}

@test "roxctl sensor generate: no overrides with collector full" {
  name="$(bundle_unique_name)"
  generate_bundle k8s "--slim-collector=false" --name "$name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_latest"
  delete_cluster "$name"
}

@test "roxctl sensor generate: with main image override. Collector should be derived from main override" {
  name="$(bundle_unique_name)"
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" --name "$name"
  assert_success
  assert_sensor_component "$out_dir" "example\.com/stackrox/main:$any_version"
  assert_collector_component "$out_dir" "example\.com/stackrox/collector:$any_version_slim"
  delete_cluster "$name"
}

@test "roxctl sensor generate: with collector override" {
  name="$(bundle_unique_name)"
  generate_bundle k8s "--collector-image-repository=example2.com/stackrox/collector" --name "$name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "example2\.com/stackrox/collector:$any_version_slim"
  delete_cluster "$name"
}

@test "roxctl sensor generate: with different overrides" {
  name="$(bundle_unique_name)"
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" "--collector-image-repository=example2.com/stackrox/collector" --name "$name"
  assert_success
  assert_sensor_component "$out_dir" "example\.com/stackrox/main:$any_version"
  assert_collector_component "$out_dir" "example2\.com/stackrox/collector:$any_version_slim"
  delete_cluster "$name"
}

@test "roxctl sensor generate: should fail if main image is provided with tag" {
  # TODO(RS-389): once we no longer accept tags in the main image this test should pass
  skip
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main:1.2.3" --name "$(bundle_unique_name)"
  assert_failure
}

@test "roxctl sensor generate: should fail if collector image is provided with tag" {
  generate_bundle k8s "--collector-image-repository=example.com/stackrox/collector:3.2.1" --name "$(bundle_unique_name)"
  assert_failure
}
