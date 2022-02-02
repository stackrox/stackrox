#!/usr/bin/env bats

load "helpers.bash"

out_dir=""
cluster_name="override-test-cluster"

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || skip "API_ENDPOINT environment variable required"
  [[ -n "$ROX_PASSWORD" ]] || skip "ROX_PASSWORD environment variable required"
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
  generate_bundle k8s --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_slim"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: no overrides with collector full" {
  generate_bundle k8s "--slim-collector=false" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_latest"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with main image override. Collector should be derived from main override" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "example\.com/stackrox/main:$any_version"
  assert_collector_component "$out_dir" "example\.com/stackrox/collector:$any_version_slim"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with collector override" {
  generate_bundle k8s "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "example2\.com/stackrox/collector:$any_version_slim"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with different overrides" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "example\.com/stackrox/main:$any_version"
  assert_collector_component "$out_dir" "example2\.com/stackrox/collector:$any_version_slim"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: should fail if main image is provided with tag" {
  # TODO(RS-389): once we no longer accept tags in the main image this test should pass
  skip
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main:1.2.3" --name "$cluster_name"
  assert_failure
}

@test "roxctl sensor generate: should fail if collector image is provided with tag" {
  generate_bundle k8s "--collector-image-repository=example.com/stackrox/collector:3.2.1" --name "$cluster_name"
  assert_failure
}
