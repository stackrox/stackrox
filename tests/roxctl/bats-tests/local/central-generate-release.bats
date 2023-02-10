#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  # remove binaries from the previous runs
  [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*

  echo "Testing roxctl version: '$(roxctl-release version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
}

setup() {
  export out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

# RELEASE / K8S

@test "roxctl-release central generate k8s should use registry.redhat.io registry" {
  run_image_defaults_registry_test roxctl-release k8s 'registry.redhat.io' 'registry.redhat.io'
}

@test "roxctl-release central generate k8s should respect customly-provided images" {
  run_image_defaults_registry_test roxctl-release k8s \
    'example.com' \
    'example.com' \
    '--main-image' 'example.com/main:1.2.3' \
    '--central-db-image' 'example.com/central-db:1.2.3' \
    '--scanner-image' 'example.com/scanner:1.2.3' \
    '--scanner-db-image' 'example.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate k8s should work when main and scanner are from different registries" {
  run_image_defaults_registry_test roxctl-release k8s \
    'example.com' \
    'example2.com' \
    '--main-image' 'example.com/main:1.2.3' \
    '--central-db-image' 'example.com/central-db:1.2.3' \
    '--scanner-image' 'example2.com/scanner:1.2.3' \
    '--scanner-db-image' 'example2.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate k8s should work when main is from custom registry and --image-defaults are used" {
  run_image_defaults_registry_test roxctl-release k8s \
    'example.com' \
    'stackrox.io' \
    '--main-image' 'example.com/main:1.2.3'  \
    '--central-db-image' 'example.com/central-db:1.2.3'\
    '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate k8s should not support --rhacs flag" {
  run_no_rhacs_flag_test roxctl-release k8s
}

@test "roxctl-release roxctl central generate k8s --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test roxctl-release k8s 'stackrox.io' 'stackrox.io' '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=rhacs should use registry.redhat.io registry" {
  run_image_defaults_registry_test roxctl-release k8s 'registry.redhat.io' 'registry.redhat.io' '--image-defaults' 'rhacs'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=development_build should use quay.io/rhacs-eng registry" {
  run_image_defaults_registry_test roxctl-release k8s 'quay.io/rhacs-eng' 'quay.io/rhacs-eng' '--image-defaults' 'development_build'
}

@test "roxctl-release roxctl central generate k8s --image-defaults=opensource should use quay.io/stackrox-io registry" {
  run_image_defaults_registry_test roxctl-release k8s 'quay.io/stackrox-io' 'quay.io/stackrox-io' '--image-defaults' 'opensource'
}

# RELEASE / OPENSHIFT

@test "roxctl-release central generate openshift should use registry.redhat.io registry" {
  run_image_defaults_registry_test roxctl-release openshift 'registry.redhat.io' 'registry.redhat.io'
}

@test "roxctl-release central generate openshift should respect customly-provided images" {
  run_image_defaults_registry_test roxctl-release openshift \
    'example.com' \
    'example.com' \
    '--main-image' 'example.com/main:1.2.3' \
    '--central-db-image' 'example.com/central-db:1.2.3' \
    '--scanner-image' 'example.com/scanner:1.2.3' \
    '--scanner-db-image' 'example.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate openshift should work when main and scanner are from different registries" {
  run_image_defaults_registry_test roxctl-release openshift \
    'example.com' \
    'example2.com' \
    '--main-image' 'example.com/main:1.2.3' \
    '--central-db-image' 'example.com/central-db:1.2.3' \
    '--scanner-image' 'example2.com/scanner:1.2.3' \
    '--scanner-db-image' 'example2.com/scanner-db:1.2.3'
}

@test "roxctl-release central generate openshift should work when main is from custom registry and --image-defaults are used" {
  run_image_defaults_registry_test roxctl-release openshift \
    'example.com' \
    'stackrox.io' \
    '--main-image' 'example.com/main:1.2.3' \
    '--central-db-image' 'example.com/central-db:1.2.3' \
    '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate openshift should not support --rhacs flag" {
  run_no_rhacs_flag_test roxctl-release openshift
}

@test "roxctl-release roxctl central generate openshift --image-defaults=stackrox.io should use stackrox.io registry" {
  run_image_defaults_registry_test roxctl-release openshift 'stackrox.io' 'stackrox.io' '--image-defaults' 'stackrox.io'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=rhacs should use registry.redhat.io registry" {
  run_image_defaults_registry_test roxctl-release openshift 'registry.redhat.io' 'registry.redhat.io' '--image-defaults' 'rhacs'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=development_build should use quay.io/rhacs-eng registry" {
  run_image_defaults_registry_test roxctl-release openshift 'quay.io/rhacs-eng' 'quay.io/rhacs-eng' '--image-defaults' 'development_build'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=opensource should use quay.io/stackrox-io registry" {
  run_image_defaults_registry_test roxctl-release openshift 'quay.io/stackrox-io' 'quay.io/stackrox-io' '--image-defaults' 'opensource'
}

@test "roxctl-release central generate k8s --debug should fail" {
  run roxctl-release central generate k8s none --output-dir "$out_dir" --debug
  assert_failure
  assert_line --regexp "ERROR:[[:space:]]+unknown flag: --debug"
}
