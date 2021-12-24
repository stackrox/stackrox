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

@test "roxctl-release central generate k8s should use stackrox.io registry" {
  run roxctl-release central generate k8s hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'stackrox.io' 'main'
  assert_components_registry "$out_dir/scanner" 'stackrox.io' 'scanner' 'scanner-db'
}

@test "roxctl-release roxctl central generate k8s --rhacs should use redhat.io registry" {
  skip_unless_rhacs
  run roxctl-release central generate --rhacs k8s hostpath --output-dir $out_dir
  assert_success
  assert_components_registry "$out_dir/central" 'registry.redhat.io' 'main'
  assert_components_registry "$out_dir/scanner" 'registry.redhat.io' 'scanner' 'scanner-db'
}
