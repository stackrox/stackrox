#!/usr/bin/env bats

load "../helpers.bash"

output_dir=""

setup_file() {
  local -r roxctl_version="$(roxctl-development version || true)"
  echo "Testing roxctl version: '${roxctl_version}'" >&3

  command -v grep || skip "Command 'grep' required."
  [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
  [[ -n "${ROX_PASSWORD}" ]] || fail "Environment variable 'ROX_PASSWORD' required"
}

setup() {
  output_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "${output_dir}"
}

scanner_generate() {
  run roxctl_authenticated scanner generate \
    --output-dir "${output_dir}" "$@"

  assert_success
}

run_scanner_generate_and_check() {
  local -r cluster_type="$1";shift

  scanner_generate --cluster-type "${cluster_type}" "$@"
  assert_success
}

@test "[openshift4] roxctl scanner generate" {
  run_scanner_generate_and_check openshift4

  assert_file_exist "${output_dir}/scanner/02-scanner-06-deployment.yaml"
  run -0 grep -q 'trusted-ca-volume' "${output_dir}/scanner/02-scanner-06-deployment.yaml"
}

@test "[openshift] roxctl scanner generate" {
  run_scanner_generate_and_check openshift

  assert_file_exist "${output_dir}/scanner/02-scanner-06-deployment.yaml"
  run -0 grep -q 'ROX_OPENSHIFT_API' "${output_dir}/scanner/02-scanner-06-deployment.yaml"
  run -1 grep -q 'trusted-ca-volume' "${output_dir}/scanner/02-scanner-06-deployment.yaml"
}

@test "[k8s] roxctl scanner generate" {
  run_scanner_generate_and_check k8s

  assert_file_exist "${output_dir}/scanner/scripts/setup.sh"
  run -0 grep -q 'KUBE_COMMAND:-kubectl' "${output_dir}/scanner/scripts/setup.sh"

  assert_file_exist "${output_dir}/scanner/02-scanner-06-deployment.yaml"
  run -1 grep -q 'ROX_OPENSHIFT_API' "${output_dir}/scanner/02-scanner-06-deployment.yaml"
}

@test "[k8s istio-support] roxctl scanner generate" {
  run_scanner_generate_and_check k8s --istio-support 1.7

  assert_file_exist "${output_dir}/scanner/02-scanner-07-service.yaml"
  run -0 grep -q "^apiVersion: networking.istio.io/v1alpha3" "${output_dir}/scanner/02-scanner-07-service.yaml"
}

@test "[k8s scanner-image] roxctl scanner generate" {
  run_scanner_generate_and_check k8s --scanner-image bats-tests

  assert_file_exist "${output_dir}/scanner/02-scanner-06-deployment.yaml"
  run -0 grep -q "bats-tests" "${output_dir}/scanner/02-scanner-06-deployment.yaml"
}
