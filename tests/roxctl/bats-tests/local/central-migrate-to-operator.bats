#!/usr/bin/env bats

load "../helpers.bash"

cr_out=""
fixtures_dir="${BATS_TEST_DIRNAME}/../testdata/central-migrate-to-operator"

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
# checked into testdata/central-migrate-to-operator/. The fixtures were
# captured once from `roxctl central generate` output before that command
# was removed (ROX-37647); they are frozen so this test no longer depends
# on it.
migrate_fixture() {
  local fixture="$1"
  run roxctl-development central migrate-to-operator --from-dir "${fixtures_dir}/${fixture}" -o "$cr_out"
  assert_success
}

# PVC storage

@test "migrate-to-operator: k8s pvc default produces claimName central-db" {
  migrate_fixture pvc-default
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "central-db"
}

@test "migrate-to-operator: k8s pvc --db-name=foo produces claimName foo" {
  migrate_fixture pvc-db-name
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "foo"
}

@test "migrate-to-operator: openshift pvc default produces claimName central-db" {
  migrate_fixture openshift-pvc-default
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "central-db"
}

@test "migrate-to-operator: openshift pvc --db-name=custom-pvc produces claimName custom-pvc" {
  migrate_fixture openshift-pvc-db-name
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "custom-pvc"
}

@test "migrate-to-operator: pvc with --db-storage-class does not include storageClassName in CR" {
  migrate_fixture pvc-storage-class
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.storageClassName' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: pvc with --db-size=200 does not include size in CR" {
  migrate_fixture pvc-db-size
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.size' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: pvc mode does not produce hostPath" {
  migrate_fixture pvc-default
  run yq e '.spec.central.db.persistence.hostPath' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: pvc mode does not produce nodeSelector" {
  migrate_fixture pvc-default
  run yq e '.spec.central.db.nodeSelector' "$cr_out"
  assert_success
  assert_output "null"
}

# Hostpath storage

@test "migrate-to-operator: k8s hostpath default produces path /var/lib/stackrox-central" {
  migrate_fixture hostpath-default
  run yq e '.spec.central.db.persistence.hostPath.path' "$cr_out"
  assert_success
  assert_output "/var/lib/stackrox-central"
}

@test "migrate-to-operator: k8s hostpath --db-hostpath=/data/db produces path /data/db" {
  migrate_fixture hostpath-custom-path
  run yq e '.spec.central.db.persistence.hostPath.path' "$cr_out"
  assert_success
  assert_output "/data/db"
}

@test "migrate-to-operator: openshift hostpath default produces path /var/lib/stackrox-central" {
  migrate_fixture openshift-hostpath-default
  run yq e '.spec.central.db.persistence.hostPath.path' "$cr_out"
  assert_success
  assert_output "/var/lib/stackrox-central"
}

@test "migrate-to-operator: hostpath mode does not produce persistentVolumeClaim" {
  migrate_fixture hostpath-default
  run yq e '.spec.central.db.persistence.persistentVolumeClaim' "$cr_out"
  assert_success
  assert_output "null"
}

# Hostpath with nodeSelector

@test "migrate-to-operator: hostpath with nodeSelector" {
  migrate_fixture hostpath-node-selector
  run yq e '.spec.central.db.nodeSelector["kubernetes.io/hostname"]' "$cr_out"
  assert_success
  assert_output "worker-1"
}

@test "migrate-to-operator: hostpath without nodeSelector produces no nodeSelector" {
  migrate_fixture hostpath-default
  run yq e '.spec.central.db.nodeSelector' "$cr_out"
  assert_success
  assert_output "null"
}

# CR metadata

@test "migrate-to-operator: produces correct apiVersion and kind" {
  migrate_fixture pvc-default
  run yq e '.apiVersion' "$cr_out"
  assert_success
  assert_output "platform.stackrox.io/v1alpha1"
  run yq e '.kind' "$cr_out"
  assert_success
  assert_output "Central"
}

@test "migrate-to-operator: produces name stackrox-central-services" {
  migrate_fixture pvc-default
  run yq e '.metadata.name' "$cr_out"
  assert_success
  assert_output "stackrox-central-services"
}

# OpenShift monitoring

@test "migrate-to-operator: openshift pvc default has no monitoring section" {
  migrate_fixture openshift-pvc-default
  run yq e '.spec.monitoring' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: openshift pvc --openshift-monitoring=true has no monitoring section" {
  migrate_fixture openshift-pvc-monitoring-true
  run yq e '.spec.monitoring' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: openshift pvc --openshift-monitoring=false sets monitoring.openshift.enabled=false" {
  migrate_fixture openshift-pvc-monitoring-false
  run yq e '.spec.monitoring.openshift.enabled' "$cr_out"
  assert_success
  assert_output "false"
}

@test "migrate-to-operator: openshift hostpath --openshift-monitoring=false sets monitoring.openshift.enabled=false" {
  migrate_fixture openshift-hostpath-monitoring-false
  run yq e '.spec.monitoring.openshift.enabled' "$cr_out"
  assert_success
  assert_output "false"
}

@test "migrate-to-operator: k8s pvc omits monitoring section entirely" {
  migrate_fixture pvc-default
  run yq e '.spec.monitoring' "$cr_out"
  assert_success
  assert_output "null"
}

# Exposure / lb-type

@test "migrate-to-operator: default has no exposure section" {
  migrate_fixture pvc-default
  run yq e '.spec.central.exposure' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: --lb-type=lb sets exposure.loadBalancer.enabled=true" {
  migrate_fixture pvc-lb-type-lb
  run yq e '.spec.central.exposure.loadBalancer.enabled' "$cr_out"
  assert_success
  assert_output "true"
}

@test "migrate-to-operator: --lb-type=np sets exposure.nodePort.enabled=true" {
  migrate_fixture pvc-lb-type-np
  run yq e '.spec.central.exposure.nodePort.enabled' "$cr_out"
  assert_success
  assert_output "true"
}

@test "migrate-to-operator: --lb-type=route sets exposure.route.enabled=true" {
  migrate_fixture openshift-pvc-lb-type-route
  run yq e '.spec.central.exposure.route.enabled' "$cr_out"
  assert_success
  assert_output "true"
}

# Plaintext endpoints

@test "migrate-to-operator: --plaintext-endpoints=8080 sets customize.envVars" {
  migrate_fixture pvc-plaintext-endpoints
  run yq e '.spec.customize.envVars[0].name' "$cr_out"
  assert_success
  assert_output "ROX_PLAINTEXT_ENDPOINTS"
  run yq e '.spec.customize.envVars[0].value' "$cr_out"
  assert_success
  assert_output "8080"
}

@test "migrate-to-operator: default has no customize section" {
  migrate_fixture pvc-default
  run yq e '.spec.customize' "$cr_out"
  assert_success
  assert_output "null"
}

# Declarative config

@test "migrate-to-operator: --declarative-config-config-maps sets declarativeConfiguration.configMaps" {
  migrate_fixture pvc-declarative-config-maps
  run yq e '.spec.central.declarativeConfiguration.configMaps[0].name' "$cr_out"
  assert_success
  assert_output "my-config"
}

@test "migrate-to-operator: --declarative-config-secrets sets declarativeConfiguration.secrets" {
  migrate_fixture pvc-declarative-secrets
  run yq e '.spec.central.declarativeConfiguration.secrets[0].name' "$cr_out"
  assert_success
  assert_output "my-secret"
}

@test "migrate-to-operator: default has no declarativeConfiguration" {
  migrate_fixture pvc-default
  run yq e '.spec.central.declarativeConfiguration' "$cr_out"
  assert_success
  assert_output "null"
}

# Default TLS cert

@test "migrate-to-operator: --default-tls-cert/key sets defaultTLSSecret" {
  migrate_fixture pvc-default-tls-cert
  run yq e '.spec.central.defaultTLSSecret.name' "$cr_out"
  assert_success
  assert_output "central-default-tls-cert"
}

@test "migrate-to-operator: default has no defaultTLSSecret" {
  migrate_fixture pvc-default
  run yq e '.spec.central.defaultTLSSecret' "$cr_out"
  assert_success
  assert_output "null"
}

# Telemetry

@test "migrate-to-operator: --enable-telemetry=false sets central.telemetry.enabled=false" {
  migrate_fixture pvc-telemetry-disabled
  run yq e '.spec.central.telemetry.enabled' "$cr_out"
  assert_success
  assert_output "false"
}

# Offline mode

@test "migrate-to-operator: --offline sets egress.connectivityPolicy=Offline" {
  migrate_fixture pvc-offline
  run yq e '.spec.egress.connectivityPolicy' "$cr_out"
  assert_success
  assert_output "Offline"
}

@test "migrate-to-operator: default has no egress section" {
  migrate_fixture pvc-default
  run yq e '.spec.egress' "$cr_out"
  assert_success
  assert_output "null"
}

# Error cases

@test "migrate-to-operator: fails without --from-dir or --namespace" {
  run roxctl-development central migrate-to-operator
  assert_failure
  assert_output --partial "at least one of the flags in the group [from-dir namespace] is required"
}

@test "migrate-to-operator: fails with --from-dir and --namespace" {
  run roxctl-development central migrate-to-operator --from-dir /tmp --namespace stackrox
  assert_failure
  assert_output --partial "if any flags in the group"
}

@test "migrate-to-operator: fails with nonexistent directory" {
  run roxctl-development central migrate-to-operator --from-dir /nonexistent/path
  assert_failure
  assert_output --partial "accessing directory"
}

@test "migrate-to-operator: fails with empty directory" {
  local empty_dir
  empty_dir="$(mktemp -d)"
  run roxctl-development central migrate-to-operator --from-dir "$empty_dir"
  assert_failure
  assert_output --partial "not found"
  rm -rf "$empty_dir"
}
