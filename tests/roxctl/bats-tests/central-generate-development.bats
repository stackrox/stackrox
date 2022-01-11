#!/usr/bin/env bats

load "helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

# DEV / K8S

@test "roxctl-development central generate k8s should use docker.io registry" {
  run_image_defaults_registry_test development k8s 'docker.io' 'docker.io' "$out_dir"
}

@test "roxctl-development central generate k8s should respect customly-provided images" {
  params=( '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example.com/scanner:1.2.3' '--scanner-db-image' 'example.com/scanner-db:1.2.3' )
  run_image_defaults_registry_test development k8s 'example.com' 'example.com' "$out_dir" "${params[@]}"
}

@test "roxctl-development central generate k8s should work when main and scanner are from different registries" {
  params=( '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example2.com/scanner:1.2.3' '--scanner-db-image' 'example2.com/scanner-db:1.2.3' )
  run_image_defaults_registry_test development k8s 'example.com' 'example2.com' "$out_dir" "${params[@]}"
}

@test "roxctl-development roxctl central generate k8s should not support --rhacs flag" {
  run_no_rhacs_flag_test development k8s
}

@test "roxctl-development roxctl central generate k8s --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test development k8s 'stackrox.io' 'stackrox.io' "$out_dir" '--image-defaults' 'stackrox.io'
}

@test "roxctl-development roxctl central generate k8s --image-defaults=rhacs should use registry.redhat.io registry" {
  run_image_defaults_registry_test development k8s 'registry.redhat.io' 'registry.redhat.io' "$out_dir" '--image-defaults' 'rhacs'
}

@test "roxctl-development roxctl central generate k8s --image-defaults=development should use docker.io registry" {
  run_image_defaults_registry_test development k8s 'docker.io' 'docker.io' "$out_dir" '--image-defaults' 'development'
}

@test "roxctl-development roxctl central generate k8s --image-defaults='' should behave as if --image-defaults would not be used" {
  run_image_defaults_registry_test development k8s 'docker.io' 'docker.io' "$out_dir" "--image-defaults=abc"
}

# RELEASE / OPENSHIFT

@test "roxctl-development central generate openshift should use docker.io registry" {
  run_image_defaults_registry_test development openshift 'docker.io' 'docker.io' "$out_dir"
}

@test "roxctl-development central generate openshift should respect customly-provided images" {
  params=( '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example.com/scanner:1.2.3' '--scanner-db-image' 'example.com/scanner-db:1.2.3' )
  run_image_defaults_registry_test development openshift 'example.com' 'example.com' "$out_dir" "${params[@]}"
}

@test "roxctl-development central generate openshift should work when main and scanner are from different registries" {
  params=( '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example2.com/scanner:1.2.3' '--scanner-db-image' 'example2.com/scanner-db:1.2.3' )
  run_image_defaults_registry_test development openshift 'example.com' 'example2.com' "$out_dir" "${params[@]}"
}

@test "roxctl-development roxctl central generate openshift should not support --rhacs flag" {
  run_no_rhacs_flag_test development openshift
}

@test "roxctl-development roxctl central generate openshift --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test development openshift 'stackrox.io' 'stackrox.io' "$out_dir" '--image-defaults' 'stackrox.io'
}

@test "roxctl-development roxctl central generate openshift --image-defaults=rhacs should use registry.redhat.io registry" {
  run_image_defaults_registry_test development openshift 'registry.redhat.io' 'registry.redhat.io' "$out_dir" '--image-defaults' 'rhacs'
}

@test "roxctl-development roxctl central generate openshift --image-defaults=development should use docker.io registry" {
  run_image_defaults_registry_test development openshift 'docker.io' 'docker.io' "$out_dir" '--image-defaults' 'development'
}

@test "roxctl-development roxctl central generate openshift --image-defaults='' should behave as if --image-defaults would not be used" {
  run_image_defaults_registry_test development openshift 'docker.io' 'docker.io' "$out_dir" "--image-defaults=abc"
}
