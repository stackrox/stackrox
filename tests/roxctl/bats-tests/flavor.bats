load "helpers.bash"

out_dir=""
original_flavor=""
port_forward_pid=""
cluster_name="flavor_cluster"

# Handle port forwards across central restarts only if not running on CI
# Disclaimer: this is necessary when running the tests locally because we
# need a running cluster and an exposed port to central pod in order to run
# the tests. Since in the CI the cluster is already provided, it's easier to
# just expect that the developer will have a running cluster accessible via
# `kubectl` rather than deploying a cluster only for the sake of local tests.
# !!!IMPORTANT!!!: Make sure to invoke `kill_port_forward` on test `teardown()`
# otherwise the tests will hand due to a background process not completed.
#handle_port_forward() {
#  if [ -z "$CI" ]; then
#    # wait for central pod to be ready
#    kubectl -n stackrox wait --timeout=180s --for=condition=ready pod -l app=central
#    run port-forward in the background and capture child pid
#    echo "Running port forward"
#    kubectl -n stackrox port-forward deploy/central 9090:8443 > /dev/null &
#    port_forward_pid="$!"
#    export API_ENDPOINT="https://localhost:9090"
#  fi
#}
#
#kill_port_forward() {
#   if [ "$port_forward_pid" != "" ]; then
#     echo "Killing port forward process"
#     kill -9 "$port_forward_pid"
#   fi
#}


setup_file() {
  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq || skip "Tests in this file require yq"
  [[ -n "$API_ENDPOINT" ]] || skip "API_ENDPOINT environment variable required"
  [[ -n "$ROX_PASSWORD" ]] || skip "ROX_PASSWORD environment variable required"
  original_flavor=$(kubectl -n stackrox exec -it deployment/central -- env | grep -i ROX_IMAGE_FLAVOR | sed 's/ROX_IMAGE_FLAVOR=//')
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
  delete_cluster "$cluster_name"
  # kill_port_forward
}

with_image_flavor() {
  flavor=$1; shift
  kubectl -n stackrox set env deployment/central ROX_IMAGE_FLAVOR="$flavor"
  kubectl -n stackrox wait --for=condition=available deployment/central --timeout=5m
  # handle_port_forward
  export_api_token
}

dev_registry_regex="docker\.io/stackrox"
stackrox_registry_regex="stackrox\.io"
collector_stackrox_registry_regex="collector\.stackrox\.io/"
any_version="[0-9]+\.[0-9]+\."
any_version_latest="[0-9]+\.[0-9]+\.[0-9]+\-latest"
any_version_slim="[0-9]+\.[0-9]+\.[0-9]+\-slim"

@test "[development_build] roxctl sensor generate defaults (slim)" {
  with_image_flavor "development_build"
  generate_bundle k8s "--slim-collector=true" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_slim"
}

@test "[development_build] roxctl sensor generate defaults (full)" {
  with_image_flavor "development_build"
  generate_bundle k8s "--slim-collector=false" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$dev_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$dev_registry_regex/collector:$any_version_latest"
}

@test "[stackrox.io] roxctl sensor generate defaults (slim)" {
  with_image_flavor "stackrox.io"
  generate_bundle k8s "--slim-collector=true" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$stackrox_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$collector_stackrox_registry_regex/collector-slim:$any_version"
}

@test "[stackrox.io] roxctl sensor generate defaults (full)" {
  with_image_flavor "stackrox.io"
  generate_bundle k8s "--slim-collector=false" --name "$cluster_name"
  assert_success
  assert_sensor_component "$out_dir" "$stackrox_registry_regex/main:$any_version"
  assert_collector_component "$out_dir" "$collector_stackrox_registry_regex/collector:$any_version"
}
