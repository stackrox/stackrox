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
  out_dir="$(mktemp -d -u)"
  # set_image_flavor "development_build"
}

teardown() {
  rm -rf "$out_dir"
  # set_image_flavor "$original_flavor"
}

@test "[k8s] roxctl sensor generate no overrides" {
  export_api_token
  generate_bundle k8s
  assert_components_registry "$out_dir" "docker.io" "sensor" "collector"
}

#@test "[k8s] roxctl sensor generate with main image override" {
#  generate_bundle k8s "--main-image-repository=example.io/rhacs"
#  assert_components_registry "$out_dir" "example.io/rhacs" "sensor" "collector"
#}
#
#@test "[k8s] roxctl sensor generate with collector override" {
#  generate_bundle k8s "--collector-image-repository=collector.example.io/rhacs"
#  assert_components_registry "$out_dir" "docker.io/stackrox" "sensor"
#  assert_components_registry "$out_dir" "collector.example.io/rhacs" "collector"
#}
#
#@test "[k8s] roxctl sensor generate with both overrides" {
# generate_bundle k8s "--main-image-repository=example.io/rhacs" "--collector-image-repository=collector.example.io/rhacs"
#  assert_components_registry "$out_dir" "example.io/rhacs" "sensor"
#  assert_components_registry "$out_dir" "collector.example.io/rhacs" "collector"
#}
#
#@test "[openshift] roxctl sensor generate no overrides" {
#  generate_bundleopenshift
#  assert_components_registry "$out_dir" "docker.io/stackrox" "sensor" "collector"
#}
#
#@test "[openshift] roxctl sensor generate with main image override" {
#  generate_bundle openshift "--main-image-repository=example.io/rhacs"
#  assert_components_registry "$out_dir" "example.io/rhacs" "sensor" "collector"
#}
#
#@test "[openshift] roxctl sensor generate with collector override" {
#  generate_bundle openshift "--collector-image-repository=collector.example.io/rhacs"
#  assert_components_registry "$out_dir" "docker.io/stackrox" "sensor"
#  assert_components_registry "$out_dir" "collector.example.io/rhacs" "collector"
#}
#
#@test "[openshift] roxctl sensor generate with both overrides" {
# generate_bundle openshift "--main-image-repository=example.io/rhacs" "--collector-image-repository=collector.example.io/rhacs"
#  assert_components_registry "$out_dir" "example.io/rhacs" "sensor"
#  assert_components_registry "$out_dir" "collector.example.io/rhacs" "collector"
#}
#
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
