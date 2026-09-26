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
  central_flavor="$(kubectl -n stackrox exec deployment/central -- env 2>&1 | grep -i '^ROX_IMAGE_FLAVOR=' | sed 's/ROX_IMAGE_FLAVOR=//' | tr -d '\r\n')"
  if [[ -z "$central_flavor" ]]; then
    echo "Failed to detect Central image flavor. Checking Central pod status:" >&3
    kubectl -n stackrox get pod -l app=central >&3 || true
    fail "Could not detect Central image flavor from deployment/central"
  fi
  echo "Detected central flavor: '$central_flavor'" >&3
  export central_flavor
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
  "rhacs")
    echo "registry.redhat.io/advanced-cluster-security"
    ;;
  *)
    fail "Unknown central flavor: '$central_flavor'"
    ;;
  esac
}

image_name_from_flavor() {
  local component="$1"
  case "$central_flavor" in
  "rhacs")
    echo "rhacs-${component}-rhel9"
    ;;
  *)
    echo "$component"
    ;;
  esac
}

any_version_regex="${any_version}[0-9x]+(-.*)?$"

@test "roxctl sensor generate: no overrides" {
  generate_bundle k8s --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/$(image_name_from_flavor main):$any_version_regex"
  assert_bundle_registry "$out_dir" "collector" "$(registry_from_flavor)/$(image_name_from_flavor collector):$any_version_regex"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with main image override. Collector should be derived from main override" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "example\.com/stackrox/main:$any_version_regex"
  assert_bundle_registry "$out_dir" "collector" "example\.com/stackrox/collector:$any_version_regex"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with collector override" {
  generate_bundle k8s "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/$(image_name_from_flavor main):$any_version_regex"
  assert_bundle_registry "$out_dir" "collector" "example2\.com/stackrox/collector:$any_version_regex"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with different overrides" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "example\.com/stackrox/main:$any_version_regex"
  assert_bundle_registry "$out_dir" "collector" "example2\.com/stackrox/collector:$any_version_regex"
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
