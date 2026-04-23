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

cleanup_cluster() {
  delete_cluster "$cluster_name"
}

@test "sensor migrate-to-operator: cluster name is detected from manifests" {
  generate_bundle_and_migrate
  run yq e '.spec.clusterName' "$cr_out"
  assert_success
  assert_output "$cluster_name"
  cleanup_cluster
}
