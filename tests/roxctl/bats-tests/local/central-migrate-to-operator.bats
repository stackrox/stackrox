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
}

teardown() {
  rm -rf "$out_dir" "$cr_out"
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
