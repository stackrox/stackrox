#!/usr/bin/env bats

load "helpers.bash"

out_dir=""
setup() {
  out_dir="$(mktemp -d -u)"
  command -v yq > /dev/null || skip "Tests in this file require yq"
  helm_output_dir="$out_dir/rendered/stackrox-central-services/templates"
  central_generate_dir="$out_dir/central"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-release central generate k8s should use stackrox.io registry" {
  run roxctl-release central generate k8s hostpath --output-dir "$out_dir"
  assert_success
  assert_output --partial "Wrote central bundle to"
  assert_components_registry "$central_generate_dir" 'stackrox.io' 'main'
}

@test "roxctl-release helm output central-services should use stackrox.io registry" {
  run roxctl-release helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"
  assert_components_registry "$helm_output_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --rhacs should use redhat.io registry" {
  run roxctl-release helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  # TODO(RS-346): Ensure that we have a proper registry address here: 'registry.redhat.io' vs 'registry.redhat.io-short'
  assert_components_registry "$helm_output_dir" 'registry.redhat.io-short' 'main' 'scanner' 'scanner-db'
}

@test "roxctl-release helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-release helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"
  assert_components_registry "$helm_output_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
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

  helm-template-central "$out_dir"
  assert_components_registry "$helm_output_dir" 'stackrox.io' 'main' 'scanner' 'scanner-db'
}
