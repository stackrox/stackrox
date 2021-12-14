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

setup_file() {
  echo "Testing roxctl version: '$(roxctl version)'" >&3
  command -v yq || skip "Tests in this file require yq"
}

setup() {
  out_dir="$(mktemp -d -u)"
  # TODO(PR): all tests are currrently skipped here to focus on helm-output first
  skip "Tests in this file are not ready yet"
}

teardown() {
  rm -rf "$out_dir"
}

@test "(devel) roxctl central generate k8s should use development registry" {
  run roxctl central generate k8s hostpath --output-dir $out_dir
  assert_success

  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/central/01-central-12-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/scanner/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/scanner/02-scanner-06-deployment.yaml"
  assert_output --regexp 'docker.io/stackrox/scanner-db:[0-9]+\.[0-9]+\.'
}

@test "(devel) roxctl central generate k8s should respect customly-provided images" {
  # Ensure that custom images are respected
  # TODO(PR): check how to set collector immage (here or somewhere else)
  run roxctl central generate k8s \
    --main-image example.com/main:1.2.3 \
    --scanner-image example.com/scanner:1.2.3 \
    --scanner-db-image example.com/scanner-db:1.2.3 \
    hostpath \
    --output-dir $out_dir
  assert_success
  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/central/01-central-12-deployment.yaml"
  assert_output 'example.com/main:1.2.3'
  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/scanner/02-scanner-06-deployment.yaml"
  assert_output 'example.com/scanner:1.2.3'
  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] |  select(.name == "db").image' "$out_dir/scanner/02-scanner-06-deployment.yaml"
  assert_output 'example.com/scanner-db:1.2.3'
}

@test "(devel) roxctl central generate k8s --rhacs should use redhat.io registry" {
  grep "\-\-rhacs" <(roxctl central generate -h) || skip "because roxctl generate does not support --rhacs flag yet"
  run roxctl central generate --rhacs k8s hostpath --output-dir $out_dir
  assert_success

  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$out_dir/central/01-central-12-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$out_dir/scanner/02-scanner-06-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-scanner:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$out_dir/scanner/02-scanner-06-deployment.yaml"
  assert_output --regexp 'registry.redhat.io/advanced-cluster-security/rhacs-rhel8-scanner-db:[0-9]+\.[0-9]+\.'
}

@test "(release) roxctl central generate k8s should use stackrox.io registry" {
  skip "Unimplemented yet"
}

@test "(release) roxctl central generate k8s --rhacs should use redhat.io registry" {
  skip "Unimplemented yet"
}
