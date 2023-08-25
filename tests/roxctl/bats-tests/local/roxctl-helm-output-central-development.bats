#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  # remove binaries from the previous runs
  [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*

  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
  chart_debug_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir" "$chart_debug_dir"
}

@test "roxctl-development helm output should support --rhacs flag" {
  run roxctl-development helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  has_deprecation_warning
}

@test "roxctl-development helm output should support --image-defaults flag" {
  run roxctl-development helm output central-services --image-defaults="stackrox.io" --output-dir "$out_dir"
  assert_success
}

@test "roxctl-development helm output help-text should state default value for --image-defaults flag" {
  run roxctl-development helm output central-services -h
  assert_success
  assert_line --regexp "--image-defaults.*\(development_build, stackrox.io, rhacs, opensource\).*default \"development_build\""
}

@test "roxctl-development helm output central-services should use quay.io/rhacs-eng registry by default" {
  run roxctl-development helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'quay.io/rhacs-eng' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-development helm output central-services --rhacs should use redhat.io registry and display deprecation warning" {
  run roxctl-development helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  has_deprecation_warning
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-development helm output central-services --image-defaults=dummy should fail" {
  run roxctl-development helm output central-services --image-defaults=dummy --output-dir "$out_dir"
  assert_failure
  assert_line --regexp "ERROR:[[:space:]]+unable to get chart meta values: '--image-defaults': unexpected value 'dummy', allowed values are \[development_build stackrox.io rhacs opensource\]"
}

@test "roxctl-development helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-development helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'stackrox.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-development helm output central-services --image-defaults=rhacs should use registry.redhat.io registry" {
  run roxctl-development helm output central-services --image-defaults=rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'registry.redhat.io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-development helm output central-services --image-defaults=development_build should use quay.io/rhacs-eng registry" {
  run roxctl-development helm output central-services --image-defaults=development_build --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'quay.io/rhacs-eng' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-development helm output central-services --image-defaults=opensource should use quay.io/stackrox-io registry" {
  run roxctl-development helm output central-services --image-defaults=opensource --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"
  assert_helm_template_central_registry "$out_dir" 'quay.io/stackrox-io' "$any_version" 'main' 'scanner' 'scanner-db'
}

@test "roxctl-development helm output central-services --rhacs --image-defaults=development_build should return error about --rhacs colliding with --image-defaults" {
  run roxctl-development helm output central-services --rhacs --image-defaults=development_build --output-dir "$out_dir"
  assert_failure
  has_deprecation_warning
  has_flag_collision_warning
}

@test "roxctl-development helm output central-services --rhacs --image-defaults=stackrox.io should return error about --rhacs colliding with --image-defaults" {
  run roxctl-development helm output central-services --rhacs --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_failure
  has_deprecation_warning
  has_flag_collision_warning
}

@test "roxctl-development helm output central-services --rhacs --image-defaults=rhacs should return error about --rhacs colliding with --image-defaults" {
  run roxctl-development helm output central-services --rhacs --image-defaults=rhacs --output-dir "$out_dir"
  assert_failure
  has_deprecation_warning
  has_flag_collision_warning
}

@test "roxctl-development helm output central-services --debug should use the local directory" {
  run_with_debug_flag_test roxctl-development helm output central-services --image-defaults=development_build --output-dir "$out_dir"
  assert_success
  assert_debug_templates_exist "$out_dir/templates"
}

@test "roxctl-development helm output central-services --debug should fail when debug dir does not exist" {
  run roxctl-development helm output central-services --image-defaults=development_build --output-dir "$out_dir" --debug --debug-path "/non-existing-dir"
  assert_failure
  assert_output --partial "no such file or directory"
}
