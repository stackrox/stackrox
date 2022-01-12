#!/usr/bin/env bats

load "helpers.bash"

out_dir=""
setup() {
  out_dir="$(mktemp -d -u)"
  command -v yq > /dev/null || skip "Tests in this file require yq"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-release helm output central-services should use stackrox.io registry" {
  run roxctl-release helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --rhacs should use redhat.io registry" {
  run roxctl-release helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-release helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=development should fail" {
  run roxctl-release helm output central-services --image-defaults=development --output-dir "$out_dir"
  assert_failure
  assert_output --partial "invalid value of '--image-defaults=development', allowed values:"
}

@test "roxctl-release helm output central-services --image-defaults='' should behave as if --image-defaults would not be used" {
  run roxctl-release helm output central-services --image-defaults='' --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --rhacs --image-defaults=stackrox.io should respect --rhacs flag, display a warning, and use redhat.io registry" {
  run roxctl-release helm output central-services --rhacs --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_line "Warning: '--rhacs' has priority over '--image-defaults'"
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' 'main' 'scanner' 'scanner-db'
}
