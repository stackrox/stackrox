#!/usr/bin/env bats

# TODO(do-not-merge): required for local and remote runs of the tests - remove before merging
test -f "/usr/lib/node_modules/bats-support/load.bash" && load "/usr/lib/node_modules/bats-support/load.bash"
test -f "/usr/lib/node_modules/bats-assert/load.bash" && load "/usr/lib/node_modules/bats-assert/load.bash"

test -f "${HOME}/bats-core/bats-support/load.bash" && load "${HOME}/bats-core/bats-support/load.bash"
test -f "${HOME}/bats-core/bats-assert/load.bash" && load "${HOME}/bats-core/bats-assert/load.bash"

# TODO(do-not-merge): uncomment before merging
# load "/usr/lib/node_modules/bats-support/load.bash"
# load "/usr/lib/node_modules/bats-assert/load.bash"

out_dir=""
setup() {
  out_dir="$(mktemp -d -u)"
  command -v yq || skip "Tests in this file require yq"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl helm output central-services should use docker.io registry" {
  run roxctl helm output central-services --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  run helm template stackrox-central-services "$out_dir" \
    -n stackrox \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"

  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl helm output central-services --rhacs should use redhat.io registry" {
  grep "\-\-rhacs" <(roxctl helm output central-services -h) || skip "because roxctl generate does not support --rhacs flag yet"
  run roxctl helm output central-services --rhacs --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart central-services to directory"

  run helm template stackrox-central-services "$out_dir" \
    -n stackrox \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"
  assert_success
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"

  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl helm output secured-cluster-services should use docker.io registry" {
  run roxctl helm output secured-cluster-services --ca /dev/null --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  run helm template stackrox-secured-cluster-services "$out_dir" \
    -n stackrox \
    --set clusterName=clusterUnderTest \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"
  assert_success

  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "roxctl helm output secured-cluster-services --rhacs should use redhat.io registry" {
  skip "Unimplemented yet. TODO: test helm-charts and K8s manifests"
  # Helm charts and yaml manifests are generated independently, so testing helm charts is not enough and we should also test the yaml manifests
  # I should get the sam results as in CircleCI, unless I run with go tag release!
}
