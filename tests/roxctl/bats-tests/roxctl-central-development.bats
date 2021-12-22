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

@test "roxctl-development helm output should support --rhacs flag" {
  run roxctl-development helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
}

@test "roxctl-development helm output should support --image-defaults flag" {
  run roxctl-development helm output central-services --image-defaults="stackrox.io" --output-dir "$out_dir"
  assert_success
}

@test "roxctl-development helm output should raise warning when both --rhacs and --image-defaults flags are used" {
  run roxctl-development helm output central-services --rhacs --image-defaults="stackrox.io" --output-dir "$out_dir"
  assert_success
  assert_output --partial "Warning: '--rhacs' has priority over '--image-defaults'"
}

@test "roxctl-development central generate k8s should use docker.io registry" {
  run roxctl-development central generate k8s hostpath --output-dir "$out_dir"
  assert_success
  assert_output --partial "Wrote central bundle to"
  assert_central_image_matches "$out_dir/central/01-central-12-deployment.yaml"    'docker\.io/stackrox/main:[0-9]+\.[0-9]+\.'
}

@test "roxctl-development helm output central-services should use docker.io registry" {
  run roxctl-development helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'docker\.io/stackrox/main:[0-9]+\.[0-9]+\.'
  assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'docker\.io/stackrox/scanner:[0-9]+\.[0-9]+\.'
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'docker\.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl-development helm output central-services --rhacs should use redhat.io registry" {
  run roxctl-development helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  # TODO(RS-380): change assertions to these three:
  # assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-main:[0-9]+\.[0-9]+\.'
  # assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-scanner:[0-9]+\.[0-9]+\.'
  # assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-scanner-db:[0-9]+\.[0-9]+\.'
  assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'docker\.io/stackrox/main:[0-9]+\.[0-9]+\.'
  assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'docker\.io/stackrox/scanner:[0-9]+\.[0-9]+\.'
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'docker\.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl-development helm output central-services --image-defaults=dummy should fail" {
  run roxctl-development helm output central-services --image-defaults=dummy --output-dir "$out_dir"
  assert_failure
  assert_output --partial "invalid value of '--image-defaults"
}

@test "roxctl-development helm output central-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-development helm output central-services --image-defaults=stackrox.io --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'stackrox\.io/main:[0-9]+\.[0-9]+\.'
  assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'stackrox\.io/scanner:[0-9]+\.[0-9]+\.'
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'stackrox\.io/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl-development helm output central-services --image-defaults=development should use docker.io registry" {
  run roxctl-development helm output central-services --image-defaults=development --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'docker\.io/stackrox/main:[0-9]+\.[0-9]+\.'
  assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'docker\.io/stackrox/scanner:[0-9]+\.[0-9]+\.'
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'docker\.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl-development helm output central-services --rhacs --image-defaults=development should respect --rhacs flag and use redhat.io registry" {
  run roxctl-development helm output central-services --rhacs --image-defaults=development --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  helm-template-central "$out_dir"

  # TODO(RS-380): change assertions to these three:
  # assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-main:[0-9]+\.[0-9]+\.'
  # assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-scanner:[0-9]+\.[0-9]+\.'
  # assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-scanner-db:[0-9]+\.[0-9]+\.'
  assert_central_image_matches "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"    'docker\.io/stackrox/main:[0-9]+\.[0-9]+\.'
  assert_scanner_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"    'docker\.io/stackrox/scanner:[0-9]+\.[0-9]+\.'
  assert_scanner_db_image_matches "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml" 'docker\.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}
