#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || fail "API_ENDPOINT environment variable required"
  [[ -n "$ROX_PASSWORD" ]] || fail "ROX_PASSWORD environment variable required"
}

setup() {
  # Cluster name must be unique
  CLUSTER_NAME="CLUSTER-bats-$BATS_SUITE_TEST_NUMBER-$(date '+%Y%m%d%H%M%S')"
  out_dir="$(mktemp -d -u)"
  # 'roxctl helm output secured-cluster-services' relies on GOPATH to find templates
  assert_file_exist "$GOPATH/src/github.com/stackrox/stackrox/image/templates/helm/stackrox-secured-cluster" || fail "Unbale to find templates. May it be a wrong GOPATH='$GOPATH'?"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-development helm output secured-cluster-services should use docker.io registry" {
  run roxctl-development helm output secured-cluster-services --remove --debug --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_args=()
  roxctl_authenticated central init-bundles fetch-ca --output "$out_dir/ca-config.yaml"
  assert_success
  yaml_valid "$out_dir/ca-config.yaml"

  # curl -sSLk \
  #   -u "admin:$ROX_PASSWORD" \
  #   -X POST \
  #   "https://${API_ENDPOINT}/v1/cluster-init/init-bundles" \
  #   -d '{"name":"deploy-'$CLUSTER'"}' | jq '.helmValuesBundle' -r | base64 --decode > "$out_dir/init-bundle.yaml"
  # assert_success
  # yaml_valid "$out_dir/init-bundle.yaml"

  roxctl_authenticated central init-bundles generate "bundle-${CLUSTER_NAME}" --output "$out_dir/cluster-init-bundle.yaml"
  assert_success
  yaml_valid "$out_dir/cluster-init-bundle.yaml"

  helm_args+=(
    -f "$out_dir/cluster-init-bundle.yaml"
    -f "$out_dir/feature-flag-values.yaml"
    -f "$out_dir/ca-config.yaml"
    -f "$out_dir/values.yaml"
    --set "clusterName=${CLUSTER_NAME}"
    --set "imagePullSecrets.allowNone=true"
  )

  run helm template stackrox-secured-cluster-services "$out_dir" \
    -n stackrox \
    "${helm_args[@]}" \
    --output-dir="$out_dir/rendered"
  assert_success

  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml"

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "collector").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_output --regexp 'docker.io/stackrox/collector:[0-9]+\.[0-9]+\.'


  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "admission-control").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "sensor").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.' || fail "sensor does not contain deployment:\n$(cat "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml")"
}


@test "roxctl-development helm output secured-cluster-services --image-defaults=development_build should use docker.io registry" {
  run roxctl-development helm output secured-cluster-services --image-defaults=development_build --remove --debug --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_args=()
  roxctl_authenticated central init-bundles fetch-ca --output "$out_dir/ca-config.yaml"
  assert_success
  yaml_valid "$out_dir/ca-config.yaml"

  roxctl_authenticated central init-bundles generate "bundle-${CLUSTER_NAME}" --output "$out_dir/cluster-init-bundle.yaml"
  assert_success
  yaml_valid "$out_dir/cluster-init-bundle.yaml"

  helm_args+=(
    -f "$out_dir/cluster-init-bundle.yaml"
    -f "$out_dir/feature-flag-values.yaml"
    -f "$out_dir/ca-config.yaml"
    -f "$out_dir/values.yaml"
    --set "clusterName=${CLUSTER_NAME}"
    --set "imagePullSecrets.allowNone=true"
  )

  run helm template stackrox-secured-cluster-services "$out_dir" \
    -n stackrox \
    "${helm_args[@]}" \
    --output-dir="$out_dir/rendered"
  assert_success

  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml"

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "collector").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_output --regexp 'docker.io/stackrox/collector:[0-9]+\.[0-9]+\.'


  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "admission-control").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "sensor").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml"
  assert_output --regexp 'docker.io/stackrox/main:[0-9]+\.[0-9]+\.' || fail "sensor does not contain deployment:\n$(cat "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml")"
}

@test "roxctl-development helm output secured-cluster-services --image-defaults=stackrox.io should use stackrox.io registry" {
  run roxctl-development helm output secured-cluster-services --image-defaults=stackrox.io --remove --debug --output-dir "$out_dir"
  assert_success
  assert_output --partial "Written Helm chart secured-cluster-services to directory"

  helm_args=()
  roxctl_authenticated central init-bundles fetch-ca --output "$out_dir/ca-config.yaml"
  assert_success
  yaml_valid "$out_dir/ca-config.yaml"

  roxctl_authenticated central init-bundles generate "bundle-${CLUSTER_NAME}" --output "$out_dir/cluster-init-bundle.yaml"
  assert_success
  yaml_valid "$out_dir/cluster-init-bundle.yaml"

  helm_args+=(
    -f "$out_dir/cluster-init-bundle.yaml"
    -f "$out_dir/feature-flag-values.yaml"
    -f "$out_dir/ca-config.yaml"
    -f "$out_dir/values.yaml"
    --set "clusterName=${CLUSTER_NAME}"
    --set "imagePullSecrets.allowNone=true"
  )

  run helm template stackrox-secured-cluster-services "$out_dir" \
    -n stackrox \
    "${helm_args[@]}" \
    --output-dir="$out_dir/rendered"
  assert_success

  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_file_exist "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml"

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "collector").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_output --regexp 'collector.stackrox.io/collector:[0-9]+\.[0-9]+\.'


  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "admission-control").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_output --regexp 'stackrox.io/main:[0-9]+\.[0-9]+\.'

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "sensor").image' "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml"
  assert_output --regexp 'stackrox.io/main:[0-9]+\.[0-9]+\.' || fail "sensor does not contain deployment:\n$(cat "$out_dir/rendered/stackrox-secured-cluster-services/templates/sensor.yaml")"
}
