#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""
original_flavor=""



setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || skip "API_ENDPOINT environment variable required"
  [[ -n "$ROX_PASSWORD" ]] || skip "ROX_PASSWORD environment variable required"
  original_flavor=$(kubectl -n stackrox exec -it deployment/central -- env | grep -i ROX_IMAGE_FLAVOR | sed 's/ROX_IMAGE_FLAVOR=//')
}

export_api_token() {
  echo "$ROX_PASSWORD $API_ENDPOINT"
  api_token="$(curl -Sskf -u "admin:$ROX_PASSWORD" "$API_ENDPOINT/v1/apitokens/generate" -d '{"name": "test", "role": "Admin"}' | jq -er .token)"
  echo "Token: $api_token"
  export ROX_API_TOKEN="$api_token"
}

set_image_flavor() {
  flavor=$1; shift
  kubectl -n stackrox set env deployment/central ROX_IMAGE_FLAVOR="$flavor"
  echo "Waiting for central pod to be ready"
  kubectl -n stackrox wait --for=condition=ready pod -l app=central
}

roxctl_cmd() {
  roxctl
}

setup() {
  # out_dir="$(mktemp -d -u)"
  out_dir="/tmp/bats/testrun-$(date '+%s')"
  mkdir -p "$out_dir"
  # set_image_flavor "development_build"
}

teardown() {
  echo "aa"
  # rm -rf "$out_dir"
  # set_image_flavor "$original_flavor"
}

dev_registry_regex="docker\.io/stackrox"
any_version="[0-9]+\.[0-9]+\."
any_version_latest="[0-9]+\.[0-9]+\.[0-9]+\-latest"
any_version_slim="[0-9]+\.[0-9]+\.[0-9]+\-slim"

@test "[k8s] roxctl sensor generate no overrides" {
  export_api_token
  generate_bundle k8s
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_slim"
}

@test "[k8s] roxctl sensor generate no overrides with collector full" {
  export_api_token
  generate_bundle k8s "--slim-collector=false"
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_latest"
}

@test "[k8s] roxctl sensor generate with main image override. Collector should be derived from main override" {
  generate_bundle k8s "--main-image-repository=example.com/stackrox/main"
  assert_sensor_component "$out_dir" "example\.com/stackrox/main:$any_version"
  assert_collector_component "$out_dir" "example\.com/stackrox/collector:$any_version_slim"
}

@test "[k8s] roxctl sensor generate with collector override" {
  generate_bundle k8s "--collector-image-repository=example2.com/stackrox/collector"
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "example2\.com/stackrox/collector:$any_version_slim"
}

@test "[k8s] roxctl sensor generate with both overrides" {
 generate_bundle k8s "--main-image-repository=example.com/stackrox/main" "--collector-image-repository=example2.com/stackrox/collector"
 assert_sensor_component "$out_dir" "example\.com/stackrox/main:$any_version"
 assert_collector_component "$out_dir" "example2\.com/stackrox/collector:$any_version_slim"
}

@test "[openshift] roxctl sensor generate no overrides" {
  generate_bundle openshift
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_slim"
}


#
#@test "[openshift] roxctl sensor generate with collector override" {
#  generate_bundle openshift "--collector-image-repository=example2.com"
#  assert_components_registry "$out_dir" "docker.io/stackrox" "sensor"
#  assert_components_registry "$out_dir" "example2.com" "collector"
#}
#
#@test "[openshift] roxctl sensor generate with both overrides" {
# generate_bundle openshift "--main-image-repository=example.com" "--collector-image-repository=collector.example.com"
#  assert_components_registry "$out_dir" "example.com" "sensor"
#  assert_components_registry "$out_dir" "collector.example.com" "collector"
#}

#@test "[stackrox.io] roxctl sensor generate with stackrox.io defaults" {
#  set_image_flavor "stackrox.io"
#  generate_bundle k8s
#  assert_components_registry "$out_dir" "stackrox.io" "sensor"
#  assert_components_registry "$out_dir" "collector.stackrox.io" "collector"
#
#}
#
#@test "[rhacs] roxctl sensor generate with rhacs defaults" {
#  set_image_flavor "rhacs"
#  generate_bundle k8s
#  assert_components_registry "$out_dir" "registry.redhat.io/advanced-cluster-security" "sensor" "collector"
#}
