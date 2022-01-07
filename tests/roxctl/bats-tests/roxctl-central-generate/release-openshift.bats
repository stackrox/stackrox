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

@test "roxctl-release central generate openshift should use stackrox.io registry" {
  run roxctl-release central generate openshift hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'stackrox.io' 'main'
  assert_components_registry "$out_dir/scanner" 'stackrox.io' 'scanner' 'scanner-db'
}

@test "roxctl-release roxctl central generate openshift should not support --rhacs flag" {
  run roxctl-release central generate --rhacs openshift hostpath --output-dir $out_dir
  assert_failure
  assert_output --partial "unknown flag: --rhacs"
  run roxctl-release central generate openshift --rhacs hostpath --output-dir $out_dir
  assert_failure
  assert_output --partial "unknown flag: --rhacs"
}

@test "roxctl-release roxctl central generate openshift --image-defaults=rhacs should use redhat.io registry" {
  skip_unless_image_defaults roxctl-release openshift
  run roxctl-release central generate openshift --image-defaults=stackrox.io hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'docker.io' 'main'
  assert_components_registry "$out_dir/scanner" 'docker.io' 'scanner' 'scanner-db'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=stackrox.io should use stackrox.io registry" {
  skip_unless_image_defaults roxctl-release openshift
  run roxctl-release central generate openshift --image-defaults=stackrox.io hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'stackrox.io' 'main'
  assert_components_registry "$out_dir/scanner" 'stackrox.io' 'scanner' 'scanner-db'
}

@test "roxctl-release roxctl central generate openshift --image-defaults=development should fail" {
  skip_unless_image_defaults roxctl-release openshift
  run roxctl-release central generate openshift --image-defaults=stackrox.io hostpath --output-dir $out_dir
  assert_failure
  assert_output --partial "invalid value of '--image-defaults=development', allowed values:"
}

@test "roxctl-release roxctl central generate openshift --image-defaults='' should behave as if --image-defaults would not be used" {
  skip_unless_image_defaults roxctl-release openshift
  run roxctl-release central generate openshift --image-defaults='' hostpath --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" 'stackrox.io' 'main'
  assert_components_registry "$out_dir/scanner" 'stackrox.io' 'scanner' 'scanner-db'
}
