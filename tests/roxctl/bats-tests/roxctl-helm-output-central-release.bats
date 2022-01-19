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

@test "roxctl-release helm output help-text should state default value for --image-defaults flag" {
  run roxctl-release helm output central-services -h
  assert_success
  assert_line --partial "default container registry for container images (default: \"rhacs\")"
}

@test "roxctl-release helm output central-services should use registry.redhat.io registry" {
  run roxctl-release helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  has_default_flavor_warning
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --rhacs should use registry.redhat.io registry and display deprecation warning" {
  run roxctl-release helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  has_deprecation_warning
  has_not_default_flavor_warning
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-release helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  has_not_default_flavor_warning
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=development_build should fail" {
  run roxctl-release helm output central-services --image-defaults=development_build --output-dir "$out_dir"
  assert_failure
  assert_output --partial "invalid value of '--image-defaults=development_build': unexpected value 'development_build', allowed values are"
}

@test "roxctl-release helm output central-services --image-defaults='' should behave as if --image-defaults would not be used" {
  run roxctl-release helm output central-services --image-defaults='' --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  has_default_flavor_warning
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --rhacs --image-defaults=stackrox.io should return error about --rhacs colliding with --image-defaults" {
  run roxctl-release helm output central-services --rhacs --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_failure
  has_deprecation_warning
  has_not_default_flavor_warning
  assert_line --partial "flag '--rhacs' collides with '--image-defaults=stackrox.io'. Remove '--rhacs' flag"
}

@test "roxctl-release helm output central-services --rhacs --image-defaults=rhacs should use redhat.io registry and display deprecation warning" {
  run roxctl-release helm output central-services --rhacs --image-defaults=rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  has_deprecation_warning
  has_not_default_flavor_warning
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' 'main' 'scanner' 'scanner-db'
}
