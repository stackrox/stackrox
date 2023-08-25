#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""
cluster_name="override-test-cluster"
central_flavor=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || fail "API_ENDPOINT environment variable required"
  [[ -n "$ROX_PASSWORD" ]] || fail "ROX_PASSWORD environment variable required"
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
    echo "docker\.io/stackrox"
    ;;
  "stackrox.io")
    echo "stackrox\.io"
    ;;
  esac
}

collector_full_from_flavor() {
   case "$central_flavor" in
   "development_build")
     echo "collector:$any_version_latest"
     ;;
   "stackrox.io")
     echo "collector:$any_version"
     ;;
   esac
}

collector_slim_from_flavor() {
    case "$central_flavor" in
     "development_build")
       echo "collector:$any_version_slim"
       ;;
     "stackrox.io")
       echo "collector-slim:$any_version"
       ;;
     esac
}

any_version_latest="${any_version}[0-9]+\-latest"
any_version_slim="${any_version}[0-9]+\-slim"

@test "roxctl sensor generate: no overrides" {
  generate_bundle k8s --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "$(registry_from_flavor)/$(collector_slim_from_flavor)"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: no overrides with collector full" {
  generate_bundle k8s "--slim-collector=false" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "$(registry_from_flavor)/$(collector_full_from_flavor)"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with main image override. Collector should be derived from main override" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "example\.com/stackrox/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "example\.com/stackrox/$(collector_slim_from_flavor)"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with collector override" {
  generate_bundle k8s "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "$(registry_from_flavor)/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "example2\.com/stackrox/$(collector_slim_from_flavor)"
  delete_cluster "$cluster_name"
}

@test "roxctl sensor generate: with different overrides" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main" "--collector-image-repository=example2.com/stackrox/collector" --name "$cluster_name"
  assert_success
  assert_bundle_registry "$out_dir" "sensor" "example\.com/stackrox/main:$any_version"
  assert_bundle_registry "$out_dir" "collector" "example2\.com/stackrox/$(collector_slim_from_flavor)"
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
