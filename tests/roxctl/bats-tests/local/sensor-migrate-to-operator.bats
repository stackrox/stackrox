#!/usr/bin/env bats

load "../helpers.bash"

cr_out=""
fixtures_dir="${BATS_TEST_DIRNAME}/../testdata/sensor-migrate-to-operator"

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
}

setup() {
  cr_out="$(mktemp -u).yaml"
}

teardown() {
  rm -f "$cr_out"
}

# migrate_fixture runs migrate-to-operator against a static manifest fixture
# checked into testdata/sensor-migrate-to-operator/. The fixtures were captured
# once from `roxctl sensor generate` output before that command was removed
# (ROX-37647); they are frozen so this test no longer depends on it.
migrate_fixture() {
  local fixture="$1"
  run roxctl-development sensor migrate-to-operator \
    --from-dir "${fixtures_dir}/${fixture}" \
    -o "$cr_out"
  assert_success
}

@test "sensor migrate-to-operator: cluster name is detected from manifests" {
  migrate_fixture default
  run yq e '.spec.clusterName' "$cr_out"
  assert_success
  assert_output "fixture-default"
}

@test "sensor migrate-to-operator: default central endpoint is omitted" {
  migrate_fixture default
  run yq e '.spec.centralEndpoint' "$cr_out"
  assert_success
  assert_output "null"
}

@test "sensor migrate-to-operator: custom central endpoint" {
  migrate_fixture custom-central-endpoint
  run yq e '.spec.centralEndpoint' "$cr_out"
  assert_success
  assert_output "my-central.example.com:443"
}

@test "sensor migrate-to-operator: enforcement disabled" {
  migrate_fixture enforcement-disabled
  run yq e '.spec.admissionControl.enforcement' "$cr_out"
  assert_success
  assert_output "Disabled"
}

@test "sensor migrate-to-operator: failure policy fail" {
  migrate_fixture fail-on-error
  run yq e '.spec.admissionControl.failurePolicy' "$cr_out"
  assert_success
  assert_output "Fail"
}

@test "sensor migrate-to-operator: collection none" {
  migrate_fixture collection-none
  run yq e '.spec.perNode.collector.collection' "$cr_out"
  assert_success
  assert_output "NoCollection"
}

@test "sensor migrate-to-operator: tolerations disabled" {
  migrate_fixture tolerations-disabled
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
