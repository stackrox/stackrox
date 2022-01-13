#!/usr/bin/env bats

load "helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-release version)'" >&3
  command -v yq || skip "Tests in this file require yq"
}

setup() {
  export out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

# RELEASE / K8S

@test "roxctl-release central generate k8s should use docker.io registry" {
  run_image_defaults_registry_test roxctl-release k8s 'stackrox.io' 'stackrox.io'
}

@test "roxctl-release central generate k8s should respect customly-provided images" {
  run_image_defaults_registry_test roxctl-release k8s \
    'example.com' \
    'example.com' \
    '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example.com/scanner:1.2.3' '--scanner-db-image' 'example.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate k8s should work when main and scanner are from different registries" {
  run_image_defaults_registry_test roxctl-release k8s \
    'example.com' \
    'example2.com' \
    '--main-image' 'example.com/main:1.2.3' \
    '--scanner-image' 'example2.com/scanner:1.2.3' \
    '--scanner-db-image' 'example2.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate k8s should work when main is from custom registry and --image-defaults are used" {
  run_image_defaults_registry_test roxctl-release k8s \
    'example.com' \
    'stackrox.io' \
    '--main-image' 'example.com/main:1.2.3' \
    '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate k8s should not support --rhacs flag" {
  run_no_rhacs_flag_test roxctl-release k8s
}

@test "roxctl-release roxctl central generate k8s --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test roxctl-release k8s 'stackrox.io' 'stackrox.io' '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=rhacs should use registry.redhat.io registry" {
  skip "RHACS flavor not ready yet" # TODO(RS-380): remove this line
  run_image_defaults_registry_test roxctl-release k8s 'registry.redhat.io' 'registry.redhat.io' '--image-defaults' 'rhacs'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=development should fail" {
  run_invalid_flavor_value_test roxctl-release k8s '--image-defaults' 'development'
}

# RELEASE / OPENSHIFT

@test "roxctl-release central generate openshift should use docker.io registry" {
  run_image_defaults_registry_test roxctl-release openshift 'stackrox.io' 'stackrox.io'
}

@test "roxctl-release central generate openshift should respect customly-provided images" {
  run_image_defaults_registry_test roxctl-release openshift \
    'example.com' \
    'example.com' \
    '--main-image' 'example.com/main:1.2.3' '--scanner-image' 'example.com/scanner:1.2.3' '--scanner-db-image' 'example.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate openshift should work when main and scanner are from different registries" {
  run_image_defaults_registry_test roxctl-release openshift \
    'example.com' \
    'example2.com' \
    '--main-image' 'example.com/main:1.2.3' \
    '--scanner-image' 'example2.com/scanner:1.2.3' \
    '--scanner-db-image' 'example2.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate openshift should work when main is from custom registry and --image-defaults are used" {
  run_image_defaults_registry_test roxctl-release openshift \
    'example.com' \
    'stackrox.io' \
    '--main-image' 'example.com/main:1.2.3' \
    '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate openshift should not support --rhacs flag" {
  run_no_rhacs_flag_test roxctl-release openshift
}

@test "roxctl-release roxctl central generate openshift --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test roxctl-release openshift 'stackrox.io' 'stackrox.io' '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=rhacs should use registry.redhat.io registry" {
  skip "RHACS flavor not ready yet" # TODO(RS-380): remove this line
  run_image_defaults_registry_test roxctl-release openshift 'registry.redhat.io' 'registry.redhat.io' '--image-defaults' 'rhacs'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=development should fail" {
  run_invalid_flavor_value_test roxctl-release openshift '--image-defaults' 'development'
}
