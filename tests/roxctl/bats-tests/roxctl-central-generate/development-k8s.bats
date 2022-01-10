#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl version)'" >&3
  command -v yq || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-development central generate k8s should use docker.io registry" {
  run roxctl-development central generate k8s pvc --output-dir "$out_dir"
  assert_success
  assert_output --partial "Wrote central bundle to"
  assert_components_registry "$out_dir/central" 'docker.io' 'main'
  assert_components_registry "$out_dir/scanner" 'docker.io' 'scanner' 'scanner-db'
}


@test "roxctl-development central generate k8s should respect customly-provided images" {
  run roxctl-development central generate k8s \
    --main-image example.com/main:1.2.3 \
    --scanner-image example.com/scanner:1.2.3 \
    --scanner-db-image example.com/scanner-db:1.2.3 \
    pvc \
    --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'example.com' 'main'
  assert_components_registry "$out_dir/scanner" 'example.com' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s should not support --rhacs flag" {
  run roxctl-development central generate --rhacs k8s pvc --output-dir "$out_dir"
  assert_failure
  assert_output --partial "unknown flag: --rhacs"
  run roxctl-development central generate k8s --rhacs pvc --output-dir "$out_dir"
  assert_failure
  assert_output --partial "unknown flag: --rhacs"
}

@test "roxctl-development roxctl central generate k8s --image-defaults=stackrox.io should use stackrox.io registry" {
  skip_unless_image_defaults roxctl-development k8s
  run roxctl-development central generate k8s --image-defaults=stackrox.io pvc --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'stackrox.io' 'main'
  assert_components_registry "$out_dir/scanner" 'stackrox.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --image-defaults=rhacs should use registry.redhat.io registry" {
  skip_unless_image_defaults roxctl-development k8s
  run roxctl-development central generate k8s --image-defaults=stackrox.io pvc --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'registry.redhat.io' 'main'
  assert_components_registry "$out_dir/scanner" 'registry.redhat.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --image-defaults=development should use docker.io registry" {
  skip_unless_image_defaults roxctl-development k8s
  run roxctl-development central generate k8s --image-defaults=development pvc --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'docker.io' 'main'
  assert_components_registry "$out_dir/scanner" 'docker.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --image-defaults='' should behave as if --image-defaults would not be used" {
  skip_unless_image_defaults roxctl-development k8s
  run roxctl-development central generate k8s --image-defaults='' pvc --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'docker.io' 'main'
  assert_components_registry "$out_dir/scanner" 'docker.io' 'scanner' 'scanner-db'
}
