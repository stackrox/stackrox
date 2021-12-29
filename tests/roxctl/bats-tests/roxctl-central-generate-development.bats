#!/usr/bin/env bats

load "helpers.bash"

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
  run roxctl-development central generate k8s hostpath --output-dir "$out_dir"
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
    hostpath \
    --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'example.com' 'main'
  assert_components_registry "$out_dir/scanner" 'example.com' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --rhacs should use redhat.io registry" {
  skip_unless_rhacs
  run roxctl-development central generate --rhacs k8s hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'registry.redhat.io' 'main'
  assert_components_registry "$out_dir/scanner" 'registry.redhat.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --image-defaults=stackrox.io should use stackrox.io registry" {
  skip_unless_image_defaults
  run roxctl-development central generate --image-defaults=stackrox.io k8s hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'stackrox.io' 'main'
  assert_components_registry "$out_dir/scanner" 'stackrox.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s should raise warning when both --rhacs and --image-defaults flags are used" {
  skip_unless_image_defaults
  skip_unless_rhacs
  run roxctl-development central generate --rhacs --image-defaults="stackrox.io" k8s hostpath --output-dir "$out_dir"
  assert_success
  assert_output --partial "Warning: '--rhacs' has priority over '--image-defaults'"
}

@test "roxctl-development roxctl central generate k8s --image-defaults=rhacs should use redhat.io registry" {
  skip_unless_image_defaults
  run roxctl-development central generate --image-defaults=stackrox.io k8s hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'redhat.io' 'main'
  assert_components_registry "$out_dir/scanner" 'redhat.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --image-defaults=development should use docker.io registry" {
  skip_unless_image_defaults
  run roxctl-development central generate --image-defaults=development k8s hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'docker.io' 'main'
  assert_components_registry "$out_dir/scanner" 'docker.io' 'scanner' 'scanner-db'
}

@test "roxctl-development roxctl central generate k8s --image-defaults='' should behave as if --image-defaults would not be used" {
  skip_unless_image_defaults
  run roxctl-development helm output central-services --image-defaults='' --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'docker.io' 'main'
  assert_components_registry "$out_dir/scanner" 'docker.io' 'scanner' 'scanner-db'
}
