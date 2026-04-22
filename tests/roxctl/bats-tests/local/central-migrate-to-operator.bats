#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""
cr_out=""

setup_file() {
  delete-outdated-binaries "$(roxctl-development version)"
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
  cr_out="$(mktemp -u).yaml"
  test_certs_dir="$(mktemp -d)"
  openssl ecparam -genkey -name prime256v1 -out "$test_certs_dir/tls-key.pem" 2>/dev/null
  openssl req -new -x509 -key "$test_certs_dir/tls-key.pem" -out "$test_certs_dir/tls-cert.pem" -days 1 -subj "/CN=test" 2>/dev/null
}

teardown() {
  rm -rf "$out_dir" "$cr_out" "$test_certs_dir"
}

generate_and_migrate() {
  run roxctl-development central generate "$@" --output-dir "$out_dir"
  assert_success
  run roxctl-development central migrate-to-operator --from-dir "$out_dir" -o "$cr_out"
  assert_success
}

# PVC storage

@test "migrate-to-operator: k8s pvc default produces claimName central-db" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "central-db"
}

@test "migrate-to-operator: k8s pvc --db-name=foo produces claimName foo" {
  generate_and_migrate k8s pvc --db-name=foo
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "foo"
}

@test "migrate-to-operator: openshift pvc default produces claimName central-db" {
  generate_and_migrate openshift pvc
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "central-db"
}

@test "migrate-to-operator: openshift pvc --db-name=custom-pvc produces claimName custom-pvc" {
  generate_and_migrate openshift pvc --db-name=custom-pvc
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.claimName' "$cr_out"
  assert_success
  assert_output "custom-pvc"
}

@test "migrate-to-operator: pvc with --db-storage-class does not include storageClassName in CR" {
  generate_and_migrate k8s pvc --db-storage-class=fast-ssd
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.storageClassName' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: pvc with --db-size=200 does not include size in CR" {
  generate_and_migrate k8s pvc --db-size=200
  run yq e '.spec.central.db.persistence.persistentVolumeClaim.size' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: pvc mode does not produce hostPath" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.db.persistence.hostPath' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: pvc mode does not produce nodeSelector" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.db.nodeSelector' "$cr_out"
  assert_success
  assert_output "null"
}

# Hostpath storage

@test "migrate-to-operator: k8s hostpath default produces path /var/lib/stackrox-central" {
  generate_and_migrate k8s hostpath
  run yq e '.spec.central.db.persistence.hostPath.path' "$cr_out"
  assert_success
  assert_output "/var/lib/stackrox-central"
}

@test "migrate-to-operator: k8s hostpath --db-hostpath=/data/db produces path /data/db" {
  generate_and_migrate k8s hostpath --db-hostpath=/data/db
  run yq e '.spec.central.db.persistence.hostPath.path' "$cr_out"
  assert_success
  assert_output "/data/db"
}

@test "migrate-to-operator: openshift hostpath default produces path /var/lib/stackrox-central" {
  generate_and_migrate openshift hostpath
  run yq e '.spec.central.db.persistence.hostPath.path' "$cr_out"
  assert_success
  assert_output "/var/lib/stackrox-central"
}

@test "migrate-to-operator: hostpath mode does not produce persistentVolumeClaim" {
  generate_and_migrate k8s hostpath
  run yq e '.spec.central.db.persistence.persistentVolumeClaim' "$cr_out"
  assert_success
  assert_output "null"
}

# Hostpath with nodeSelector

@test "migrate-to-operator: hostpath with nodeSelector" {
  generate_and_migrate k8s hostpath \
    --db-node-selector-key=kubernetes.io/hostname \
    --db-node-selector-value=worker-1
  run yq e '.spec.central.db.nodeSelector["kubernetes.io/hostname"]' "$cr_out"
  assert_success
  assert_output "worker-1"
}

@test "migrate-to-operator: hostpath without nodeSelector produces no nodeSelector" {
  generate_and_migrate k8s hostpath
  run yq e '.spec.central.db.nodeSelector' "$cr_out"
  assert_success
  assert_output "null"
}

# CR metadata

@test "migrate-to-operator: produces correct apiVersion and kind" {
  generate_and_migrate k8s pvc
  run yq e '.apiVersion' "$cr_out"
  assert_success
  assert_output "platform.stackrox.io/v1alpha1"
  run yq e '.kind' "$cr_out"
  assert_success
  assert_output "Central"
}

@test "migrate-to-operator: produces name stackrox-central-services" {
  generate_and_migrate k8s pvc
  run yq e '.metadata.name' "$cr_out"
  assert_success
  assert_output "stackrox-central-services"
}

# OpenShift monitoring

@test "migrate-to-operator: openshift pvc default has no monitoring section" {
  generate_and_migrate openshift pvc
  run yq e '.spec.monitoring' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: openshift pvc --openshift-monitoring=false sets monitoring.openshift.enabled=false" {
  generate_and_migrate openshift pvc --openshift-monitoring=false
  run yq e '.spec.monitoring.openshift.enabled' "$cr_out"
  assert_success
  assert_output "false"
}

@test "migrate-to-operator: openshift hostpath --openshift-monitoring=false sets monitoring.openshift.enabled=false" {
  generate_and_migrate openshift hostpath --openshift-monitoring=false
  run yq e '.spec.monitoring.openshift.enabled' "$cr_out"
  assert_success
  assert_output "false"
}

@test "migrate-to-operator: k8s pvc omits monitoring section entirely" {
  generate_and_migrate k8s pvc
  run yq e '.spec.monitoring' "$cr_out"
  assert_success
  assert_output "null"
}

# Exposure / lb-type

@test "migrate-to-operator: default has no exposure section" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.exposure' "$cr_out"
  assert_success
  assert_output "null"
}

@test "migrate-to-operator: --lb-type=lb sets exposure.loadBalancer.enabled=true" {
  generate_and_migrate k8s pvc --lb-type=lb
  run yq e '.spec.central.exposure.loadBalancer.enabled' "$cr_out"
  assert_success
  assert_output "true"
}

@test "migrate-to-operator: --lb-type=np sets exposure.nodePort.enabled=true" {
  generate_and_migrate k8s pvc --lb-type=np
  run yq e '.spec.central.exposure.nodePort.enabled' "$cr_out"
  assert_success
  assert_output "true"
}

@test "migrate-to-operator: --lb-type=route sets exposure.route.enabled=true" {
  generate_and_migrate openshift pvc --lb-type=route
  run yq e '.spec.central.exposure.route.enabled' "$cr_out"
  assert_success
  assert_output "true"
}

# Plaintext endpoints

@test "migrate-to-operator: --plaintext-endpoints=8080 sets customize.envVars" {
  generate_and_migrate k8s pvc --plaintext-endpoints=8080
  run yq e '.spec.customize.envVars[0].name' "$cr_out"
  assert_success
  assert_output "ROX_PLAINTEXT_ENDPOINTS"
  run yq e '.spec.customize.envVars[0].value' "$cr_out"
  assert_success
  assert_output "8080"
}

@test "migrate-to-operator: default has no customize section" {
  generate_and_migrate k8s pvc
  run yq e '.spec.customize' "$cr_out"
  assert_success
  assert_output "null"
}

# Declarative config

@test "migrate-to-operator: --declarative-config-config-maps sets declarativeConfiguration.configMaps" {
  generate_and_migrate k8s pvc --declarative-config-config-maps=my-config
  run yq e '.spec.central.declarativeConfiguration.configMaps[0].name' "$cr_out"
  assert_success
  assert_output "my-config"
}

@test "migrate-to-operator: --declarative-config-secrets sets declarativeConfiguration.secrets" {
  generate_and_migrate k8s pvc --declarative-config-secrets=my-secret
  run yq e '.spec.central.declarativeConfiguration.secrets[0].name' "$cr_out"
  assert_success
  assert_output "my-secret"
}

@test "migrate-to-operator: default has no declarativeConfiguration" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.declarativeConfiguration' "$cr_out"
  assert_success
  assert_output "null"
}

# Default TLS cert

@test "migrate-to-operator: --default-tls-cert/key sets defaultTLSSecret" {
  generate_and_migrate k8s pvc \
    --default-tls-cert="$test_certs_dir/tls-cert.pem" \
    --default-tls-key="$test_certs_dir/tls-key.pem"
  run yq e '.spec.central.defaultTLSSecret.name' "$cr_out"
  assert_success
  assert_output "central-default-tls-cert"
}

@test "migrate-to-operator: default has no defaultTLSSecret" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.defaultTLSSecret' "$cr_out"
  assert_success
  assert_output "null"
}

# Telemetry

@test "migrate-to-operator: --enable-telemetry=false sets central.telemetry.enabled=false" {
  generate_and_migrate k8s pvc --enable-telemetry=false
  run yq e '.spec.central.telemetry.enabled' "$cr_out"
  assert_success
  assert_output "false"
}

@test "migrate-to-operator: default has no telemetry section" {
  generate_and_migrate k8s pvc
  run yq e '.spec.central.telemetry' "$cr_out"
  assert_success
  assert_output "null"
}

# Offline mode

@test "migrate-to-operator: --offline sets egress.connectivityPolicy=Offline" {
  generate_and_migrate k8s pvc --offline
  run yq e '.spec.egress.connectivityPolicy' "$cr_out"
  assert_success
  assert_output "Offline"
}

@test "migrate-to-operator: default has no egress section" {
  generate_and_migrate k8s pvc
  run yq e '.spec.egress' "$cr_out"
  assert_success
  assert_output "null"
}

# Error cases

@test "migrate-to-operator: fails without --from-dir or --namespace" {
  run roxctl-development central migrate-to-operator
  assert_failure
  assert_output --partial "either --from-dir or --namespace must be specified"
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
