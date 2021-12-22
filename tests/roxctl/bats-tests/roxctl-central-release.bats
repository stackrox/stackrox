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

@test "roxctl-release central generate k8s should use stackrox.io registry" {
  run roxctl-release central generate k8s hostpath --output-dir "$out_dir"
  assert_success
  assert_output --partial "Wrote central bundle to"

  assert_central_image_matches "$out_dir/central/01-central-12-deployment.yaml" 'stackrox.io/main:[0-9]+\.[0-9]+\.'
}

@test "roxctl-release helm output central-services should use stackrox.io registry" {
  run roxctl-release helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  reg_prefix="stackrox.io/"
  assert_central_image_matches    "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml" "${reg_prefix}main:[0-9]+\.[0-9]+\."
  assert_scanner_image_matches    "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner:[0-9]+\.[0-9]+\."
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner-db:[0-9]+\.[0-9]+\."
}

@test "roxctl-release helm output central-services --rhacs should use redhat.io registry" {
  run roxctl-release helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  reg_prefix="registry.redhat.io/advanced-cluster-security/rhacs-rhel8-" # should be?
  reg_prefix="registry.redhat.io/rh-acs/" # is

  assert_central_image_matches    "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml" "${reg_prefix}main:[0-9]+\.[0-9]+\."
  assert_scanner_image_matches    "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner:[0-9]+\.[0-9]+\."
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner-db:[0-9]+\.[0-9]+\."
}

@test "roxctl-release helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-release helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  reg_prefix="stackrox.io/"
  assert_central_image_matches    "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml" "${reg_prefix}main:[0-9]+\.[0-9]+\."
  assert_scanner_image_matches    "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner:[0-9]+\.[0-9]+\."
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner-db:[0-9]+\.[0-9]+\."
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

  reg_prefix="stackrox.io/"
  assert_central_image_matches    "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml" "${reg_prefix}main:[0-9]+\.[0-9]+\."
  assert_scanner_image_matches    "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner:[0-9]+\.[0-9]+\."
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" "${reg_prefix}scanner-db:[0-9]+\.[0-9]+\."
}
