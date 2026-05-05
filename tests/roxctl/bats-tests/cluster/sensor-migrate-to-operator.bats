#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""
cr_out=""
cluster_name="sensor-migrate-e2e"

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || fail "API_ENDPOINT environment variable required"
  [[ -n "$ROX_ADMIN_PASSWORD" ]] || fail "ROX_ADMIN_PASSWORD environment variable required"
}

setup() {
  out_dir="$(mktemp -d -u)"
  cr_out="$(mktemp -u).yaml"
}

teardown() {
  delete_cluster "$cluster_name" 2>/dev/null || true
  rm -rf "$out_dir" "$cr_out"
}

generate_bundle_and_migrate() {
  generate_bundle k8s --name "$cluster_name" "$@"
  assert_success
  run roxctl-development sensor migrate-to-operator \
    --from-dir "$out_dir" \
    -o "$cr_out"
  assert_success
}

@test "sensor migrate-to-operator: cluster name is detected from manifests" {
  generate_bundle_and_migrate
  run yq e '.spec.clusterName' "$cr_out"
  assert_success
  assert_output "$cluster_name"
}

@test "sensor migrate-to-operator: default central endpoint is omitted" {
  generate_bundle_and_migrate
  run yq e '.spec.centralEndpoint' "$cr_out"
  assert_success
  assert_output "null"
}

@test "sensor migrate-to-operator: custom central endpoint" {
  generate_bundle_and_migrate --central=my-central.example.com:443
  run yq e '.spec.centralEndpoint' "$cr_out"
  assert_success
  assert_output "my-central.example.com:443"
}

@test "sensor migrate-to-operator: enforcement disabled" {
  generate_bundle_and_migrate --admission-controller-enforcement=false
  run yq e '.spec.admissionControl.enforcement' "$cr_out"
  assert_success
  assert_output "Disabled"
}

@test "sensor migrate-to-operator: failure policy fail" {
  generate_bundle_and_migrate --admission-controller-fail-on-error
  run yq e '.spec.admissionControl.failurePolicy' "$cr_out"
  assert_success
  assert_output "Fail"
}

@test "sensor migrate-to-operator: collection none" {
  generate_bundle_and_migrate --collection-method=none
  run yq e '.spec.perNode.collector.collection' "$cr_out"
  assert_success
  assert_output "NoCollection"
}

@test "sensor migrate-to-operator: tolerations disabled" {
  generate_bundle_and_migrate --disable-tolerations
  run yq e '.spec.perNode.taintToleration' "$cr_out"
  assert_success
  assert_output "AvoidTaints"
}

# Error cases

@test "sensor migrate-to-operator: fails without --from-dir or --namespace" {
  run roxctl-development sensor migrate-to-operator
  assert_failure
  assert_output --partial "at least one of the flags in the group [from-dir namespace] is required"
}

@test "sensor migrate-to-operator: fails with --from-dir and --namespace" {
  run roxctl-development sensor migrate-to-operator --from-dir /tmp --namespace stackrox
  assert_failure
  assert_output --partial "if any flags in the group"
}

@test "sensor migrate-to-operator: fails with nonexistent directory" {
  run roxctl-development sensor migrate-to-operator --from-dir /nonexistent/path
  assert_failure
  assert_output --partial "accessing directory"
}

@test "sensor migrate-to-operator: fails with empty directory" {
  local empty_dir
  empty_dir="$(mktemp -d)"
  run roxctl-development sensor migrate-to-operator --from-dir "$empty_dir"
  assert_failure
  assert_output --partial "not found"
  rm -rf "$empty_dir"
}
