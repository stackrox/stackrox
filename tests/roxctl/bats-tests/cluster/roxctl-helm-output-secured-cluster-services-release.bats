#!/usr/bin/env bats

load "helpers.bash"

out_dir=""
setup() {
  out_dir="$(mktemp -d -u)"
  command -v yq || skip "Tests in this file require yq"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-release helm output secured-cluster-services --rhacs should use redhat.io registry" {
  run roxctl-release helm output secured-cluster-services --ca "${ca_cert}-cert.pem" --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  run helm template stackrox-secured-cluster-services "$out_dir" \
    -n stackrox \
    --set clusterName=clusterUnderTest \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"
  assert_success

  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-scanner-db:[0-9]+\.[0-9]+\.'
}
