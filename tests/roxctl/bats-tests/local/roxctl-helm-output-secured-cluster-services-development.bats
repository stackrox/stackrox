#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  # remove binaries from the previous runs
  delete-outdated-binaries "$(roxctl-development version)"

  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  assert_file_exist "$test_data/helm-output-secured-cluster-services/ca-config.yaml"
  assert_file_exist "$test_data/helm-output-secured-cluster-services/cluster-init-bundle.yaml"
}

setup() {
  CLUSTER_NAME="CLUSTER-1"
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-development helm output secured-cluster-services (default flavor) should use quay.io/rhacs-eng registry" {
  # Running this with --debug flag would cause reading the helm templates from "$GOPATH/src/github.com/stackrox/stackrox/image/templates/helm/stackrox-secured-cluster"
  run roxctl-development helm output secured-cluster-services --remove --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_template_secured_cluster "$out_dir" "$out_dir/rendered" "$CLUSTER_NAME"
  assert_components_registry "$out_dir/rendered" "quay.io/rhacs-eng" "$any_version" 'collector' 'admission-controller' 'sensor'
}

@test "roxctl-development helm output secured-cluster-services --image-defaults=development_build should use quay.io/rhacs-eng registry" {
  run roxctl-development helm output secured-cluster-services --image-defaults=development_build --remove --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_template_secured_cluster "$out_dir" "$out_dir/rendered" "$CLUSTER_NAME"
  assert_components_registry "$out_dir/rendered" "quay.io/rhacs-eng" "$any_version" 'collector' 'admission-controller' 'sensor'
}

@test "roxctl-development helm output secured-cluster-services --image-defaults=rhacs should use registry.redhat.io registry (default collector)" {
  run roxctl-development helm output secured-cluster-services --image-defaults=rhacs --remove --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_template_secured_cluster "$out_dir" "$out_dir/rendered" "$CLUSTER_NAME"
  assert_components_registry "$out_dir/rendered" "registry.redhat.io" "$any_version" 'collector' 'admission-controller' 'sensor'
}

@test "roxctl-development helm output secured-cluster-services --image-defaults=opensource should use quay.io/stackrox-io registry" {
  run roxctl-development helm output secured-cluster-services --image-defaults=opensource --remove --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_template_secured_cluster "$out_dir" "$out_dir/rendered" "$CLUSTER_NAME"
  assert_components_registry "$out_dir/rendered" "quay.io/stackrox-io" "$any_version" 'collector' 'admission-controller' 'sensor'
}
