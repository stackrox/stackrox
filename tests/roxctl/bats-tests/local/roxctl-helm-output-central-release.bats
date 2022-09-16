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
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-release helm output help-text should state default value for --image-defaults flag" {
  run roxctl-release helm output central-services -h
  assert_success
  assert_line --regexp "--image-defaults.*\(rhacs, opensource\).*default \"rhacs\""
}

@test "roxctl-release helm output central-services should use registry.redhat.io registry by default" {
  run roxctl-release helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --rhacs should use registry.redhat.io registry and display deprecation warning" {
  run roxctl-release helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  has_deprecation_warning
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-release helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'stackrox.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=rhacs should use registry.redhat.io registry" {
  run roxctl-release helm output central-services --image-defaults=rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=development_build should use quay.io/rhacs-eng registry" {
  run roxctl-release helm output central-services --image-defaults=development_build --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'quay.io/rhacs-eng' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=opensource should use quay.io/stackrox-io registry" {
  run roxctl-release helm output central-services --image-defaults=opensource --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'quay.io/stackrox-io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults='' should fail with unexpected value of --image-defaults" {
  run roxctl-release helm output central-services --image-defaults='' --output-dir "$out_dir"
  assert_failure
  assert_line --regexp "ERROR:[[:space:]]+unable to get chart meta values: '--image-defaults': unexpected value '', allowed values are \[rhacs opensource\]"
}

@test "roxctl-release helm output central-services --rhacs --image-defaults=stackrox.io should return error about --rhacs colliding with --image-defaults" {
  run roxctl-release helm output central-services --rhacs --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_failure
  has_deprecation_warning
  has_flag_collision_warning
}

@test "roxctl-release helm output central-services --rhacs --image-defaults=rhacs should use redhat.io registry and display deprecation warning" {
  run roxctl-release helm output central-services --rhacs --image-defaults=rhacs --output-dir "$out_dir"
  assert_failure
  has_deprecation_warning
  has_flag_collision_warning
}

@test "roxctl-release helm output central-services --debug should fail" {
  run roxctl-release helm output central-services --image-defaults=development_build --output-dir "$out_dir" --debug
  assert_failure
  assert_line --regexp "ERROR:[[:space:]]+unknown flag: --debug"
}
