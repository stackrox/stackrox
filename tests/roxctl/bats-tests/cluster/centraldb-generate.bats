#!/usr/bin/env bats

load "../helpers.bash"

output_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3

  command -v cat || skip "Command 'cat' required."
  command -v grep || skip "Command 'grep' required."
  [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
  [[ -n "${ROX_PASSWORD}" ]] || fail "Environment variable 'ROX_PASSWORD' required"
  bats_require_minimum_version 1.5.0 # FIXME: Remove before merging
}

setup() {
  output_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "${output_dir}"
}

central_db_generate() {
  run roxctl_authenticated central db generate --output-dir "${output_dir}" "$@"

  assert_success
}

run_central_db_generate_and_check() {
  local -r cluster_type="$1";shift

  central_db_generate "${cluster_type}" "$@"
  assert_success
}

assert_number_of_resources() {
    local -r cluster_type="$1"; shift
    local expected="$1"; shift
    local resources_count=$(cat "${output_dir}/central/"*.yaml | grep -c "^apiVersion") || true

    # On OpenShift clusters, we add a Role+RoleBinding with SCCs to our deployments, hence add 2 resources
    if [ "${cluster_type}" = "openshift" ]; then
            expected=$((${expected} + 2))
    fi

    [[ "${resources_count}" = "${expected}" ]] || fail "Unexpected number of resources, expected ${expected} actual ${resources_count}"
}

assert_essential_files() {
  assert_file_exist "${output_dir}/README"
  assert_file_exist "${output_dir}/scripts/setup.sh"
  assert_file_exist "${output_dir}/scripts/docker-auth.sh"
  assert_file_exist "${output_dir}/central/01-central-00-db-serviceaccount.yaml"
  assert_file_exist "${output_dir}/central/01-central-05-db-tls-secret.yaml"
  assert_file_exist "${output_dir}/central/01-central-08-db-configmap.yaml"
  assert_file_exist "${output_dir}/central/01-central-08-external-db-configmap.yaml"
  assert_file_exist "${output_dir}/central/01-central-10-db-networkpolicy.yaml"
  assert_file_exist "${output_dir}/central/01-central-12-central-db.yaml"
  if [ "${cluster_type}" = "openshift" ]; then
    assert_file_exist "${output_dir}/central/01-central-02-db-security.yaml"
  fi
}

test_generate_hostpath() {
  local -r cluster_type="$1";shift
  run_central_db_generate_and_check "${cluster_type}" hostpath

  assert_essential_files
  assert_number_of_resources "${cluster_type}" 11

  run -0 grep -q "mountPath: /var/lib/postgresql/data" "${output_dir}/central/01-central-12-central-db.yaml"
}

test_generate_pvc() {
  local -r cluster_type="$1";shift
  run_central_db_generate_and_check "${cluster_type}" pvc

  assert_essential_files
  assert_number_of_resources "${cluster_type}" 12
  assert_file_exist "${output_dir}/central/01-central-11-db-pvc.yaml"
  run -0 grep -q "kind: PersistentVolumeClaim" "${output_dir}/central/01-central-11-db-pvc.yaml"
  run -0 grep -q "claimName: central-db" "${output_dir}/central/01-central-12-central-db.yaml"
}

test_generate_with_image() {
  local -r cluster_type="$1";shift
  run_central_db_generate_and_check "${cluster_type}" none --central-db-image bats-tests

  assert_essential_files
  assert_number_of_resources "${cluster_type}" 11
  run -0 grep -q "bats-tests" "${output_dir}/central/01-central-12-central-db.yaml"
}

test_generate_with_image_default() {
  local -r cluster_type="$1";shift
  run_central_db_generate_and_check "${cluster_type}" none --image-defaults=opensource

  assert_essential_files
  assert_number_of_resources "${cluster_type}" 11
  assert_file_exist "${output_dir}/central/01-central-02-db-psps.yaml"
  run -0 grep -q "image: \"quay.io/stackrox-io" "${output_dir}/central/01-central-12-central-db.yaml"
}

test_generate_security_policies_false() {
  local -r cluster_type="$1";shift
  run_central_db_generate_and_check "${cluster_type}" none --enable-pod-security-policies=false

  assert_essential_files
  assert_number_of_resources "${cluster_type}" 8
  assert_file_not_exist "${output_dir}/central/01-central-02-db-psps.yaml"
  assert_file_not_exist "${output_dir}/central/01-central-11-db-pvc.yaml"
}

test_generate_default() {
  local -r cluster_type="$1";shift
  run_central_db_generate_and_check "${cluster_type}" none

  assert_essential_files
  assert_number_of_resources "${cluster_type}" 11
  assert_file_exist "${output_dir}/central/01-central-02-db-psps.yaml"
  assert_file_not_exist "${output_dir}/central/01-central-11-db-pvc.yaml"
  run -0 grep -q "image: \"quay.io/rhacs-eng" "${output_dir}/central/01-central-12-central-db.yaml"
}

# K8s tests
@test "[central-db-bundle k8s] roxctl central db generate hostpath" {
    test_generate_hostpath k8s
}

@test "[central-db-bundle k8s] roxctl central db generate k8s pvc" {
    test_generate_pvc k8s
}

@test "[central-db-bundle k8s] roxctl central db generate with image" {
    test_generate_with_image k8s
}

@test "[central-db-bundle k8s] roxctl central db generate image-defaults opensource" {
    test_generate_with_image_default k8s
}

@test "[central-db-bundle k8s] roxctl central db generate enable-pod-security-policies false" {
    test_generate_security_policies_false k8s
}

@test "[central-db-bundle k8s] roxctl central db generate default" {
    test_generate_default k8s
}

# Openshift tests
@test "[central-db-bundle openshift] roxctl central db generate hostpath" {
    test_generate_hostpath openshift
}

@test "[central-db-bundle openshift] roxctl central db generate openshift pvc" {
    test_generate_pvc openshift
}

@test "[central-db-bundle openshift] roxctl central db generate with image" {
    test_generate_with_image openshift
}

@test "[central-db-bundle openshift] roxctl central db generate image-defaults opensource" {
    test_generate_with_image_default openshift
}

@test "[central-db-bundle openshift] roxctl central db generate enable-pod-security-policies false" {
    test_generate_security_policies_false openshift
}

@test "[central-db-bundle openshift] roxctl central db generate default" {
    test_generate_default openshift
}
