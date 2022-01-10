#!/usr/bin/env bats

load "helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-release version)'" >&3
  command -v yq || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

# DEV / K8S

@test "roxctl-release central generate k8s should use docker.io registry" {
  run_image_defaults_registry_test release k8s 'stackrox.io' "$out_dir"
}

@test "roxctl-release central generate k8s should respect customly-provided images" {
  params=( '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example.com/scanner:1.2.3' '--scanner-db-image' 'example.com/scanner-db:1.2.3' )
  run_image_defaults_registry_test release k8s 'example.com' "$out_dir" "${params[@]}"
}

@test "roxctl-release roxctl central generate k8s should not support --rhacs flag" {
  run_no_rhacs_flag_test release k8s
}

@test "roxctl-release roxctl central generate k8s --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test release k8s 'stackrox.io' "$out_dir" '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=rhacs should use registry.redhat.io registry" {
  run_image_defaults_registry_test release k8s 'registry.redhat.io' "$out_dir" '--image-defaults' 'rhacs'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=development should fail" {
  run_invalid_flavor_value_test release k8s '--image-defaults' 'development'
}

@test "roxctl-release roxctl central generate k8s --image-defaults='' should behave as if --image-defaults would not be used" {
  run_image_defaults_registry_test release k8s 'stackrox.io' "$out_dir" "--image-defaults=abc"
}

# RELEASE / OPENSHIFT

@test "roxctl-release central generate openshift should use docker.io registry" {
  run_image_defaults_registry_test release openshift 'stackrox.io' "$out_dir"
}

@test "roxctl-release central generate openshift should respect customly-provided images" {
  params=( '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example.com/scanner:1.2.3' '--scanner-db-image' 'example.com/scanner-db:1.2.3' )
  run_image_defaults_registry_test release openshift 'example.com' "$out_dir" "${params[@]}"
}

@test "roxctl-release roxctl central generate openshift should not support --rhacs flag" {
  run_no_rhacs_flag_test release openshift
}

@test "roxctl-release roxctl central generate openshift --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test release openshift 'stackrox.io' "$out_dir" '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=rhacs should use registry.redhat.io registry" {
  run_image_defaults_registry_test release openshift 'registry.redhat.io' "$out_dir" '--image-defaults' 'rhacs'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=development should fail" {
  run_invalid_flavor_value_test release openshift '--image-defaults' 'development'
}

@test "roxctl-release roxctl central generate openshift --image-defaults='' should behave as if --image-defaults would not be used" {
  run_image_defaults_registry_test release openshift 'stackrox.io' "$out_dir" "--image-defaults=abc"
}
