#!/usr/bin/env bats

load "helpers.bash"

out_dir=""
setup() {
  out_dir="$(mktemp -d -u)"
  command -v yq || skip "Tests in this file require yq"

  ca_cert="$(mktemp -u)"
  openssl genrsa 2048 > "${ca_cert}-key.pem"
  openssl req -new -x509 -nodes -days 365000 \
    -subj "/C=US/ST=California/L=San Francisco/O=RedHat/OU=Maple/CN=localhost" \
    -key "${ca_cert}-key.pem" \
    -out "${ca_cert}-cert.pem"
}

teardown() {
  rm -rf "$out_dir"
  rm -f "${ca_cert}-key.pem" "${ca_cert}-cert.pem"
}

@test "(devel) roxctl helm output secured-cluster-services should use docker.io registry" {
  run roxctl helm output secured-cluster-services --ca "${ca_cert}-cert.pem" --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  skip "Generating CA not ready yet"

  run helm template stackrox-secured-cluster-services "$out_dir" \
    --debug \
    -n stackrox \
    --set clusterName=clusterUnderTest \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"
  # This will fail with error 'A CA certificate must be specified', but the --debug flag would allow to generate some yamls
  assert_failure

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}
