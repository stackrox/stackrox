#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""
cluster_name="override-test-cluster"
central_flavor=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || fail "API_ENDPOINT environment variable required"
  [[ -n "$ROX_ADMIN_PASSWORD" ]] || fail "ROX_ADMIN_PASSWORD environment variable required"
  central_flavor="$(kubectl -n stackrox exec -it deployment/central -- env | grep -i ROX_IMAGE_FLAVOR | sed 's/ROX_IMAGE_FLAVOR=//')"
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

registry_from_flavor() {
  case "$central_flavor" in
  "development_build")
    echo "quay.io/rhacs-eng"
    ;;
  "opensource")
    echo "quay.io/stackrox-io"
    ;;
  esac
}

any_version_latest="${any_version}[0-9]+\-latest"
any_version_slim="${any_version}[0-9]+\-slim"

@test "roxctl sensor generate: no overrides" {
  generate_bundle k8s --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "$(registry_from_flavor)/collector:$any_version"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with main image override. Collector should be derived from main override" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "example\.com/stackrox/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "example\.com/stackrox/collector:$any_version"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with collector override" {
  generate_bundle k8s "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "example2\.com/stackrox/collector:$any_version"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with different overrides" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "example\.com/stackrox/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "example2\.com/stackrox/collector:$any_version"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: should fail if main image is provided with tag" {
  skip "#TODO(RS-389): once we no longer accept tags in the main image this test should pass"
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main:1.2.3" --name "$cluster_name"
  assert_failure
}

@test "roxctl sensor generate: should fail if collector image is provided with tag" {
  generate_bundle k8s "--collector-image-repository=example.com/stackrox/collector:3.2.1" --name "$cluster_name"
  assert_failure
}

@test "roxctl sensor generate: should succeed if collector slim is requested" {
  generate_bundle k8s "--slim-collector=true" --name "$cluster_name"
  assert_success
  assert_line --partial "The --slim-collector flag has been deprecated and will be removed in future versions of roxctl. It will be ignored from version 4.7 onwards."
  assert_bundle_registry "$out_dir" "collector" "$(registry_from_flavor)/collector:$any_version"
  delete_cluster "$cluster_name"
}
