#!/usr/bin/env bash

# Allow to run the tests locally provided that bats-helpers are installed in $HOME/bats-core
bats_helpers_root="${HOME}/bats-core"
if [[ ! -f "${bats_helpers_root}/bats-support/load.bash" ]]; then
  # Location of bats-helpers in the CI image
  bats_helpers_root="/usr/lib/node_modules"
fi
load "${bats_helpers_root}/bats-support/load.bash"
load "${bats_helpers_root}/bats-assert/load.bash"

# luname outputs uname in lowercase
luname() {
  uname | tr '[:upper:]' '[:lower:]'
}

tmp_roxctl="tmp/roxctl-bats/bin"

# roxctl-development-cmd prints the path to roxctl built with GOTAGS=''. It builds the binary if needed
roxctl-development-cmd() {
  if [[ ! -x "${tmp_roxctl}/roxctl-dev" ]]; then
    _uname="$(luname)"
    mkdir -p "$tmp_roxctl"
    make -s "cli-${_uname}" GOTAGS='' 2>&3
    mv "bin/${_uname}/roxctl" "${tmp_roxctl}/roxctl-dev"
  fi
  echo "${tmp_roxctl}/roxctl-dev"
}

# roxctl-development runs roxctl built with GOTAGS=''. It builds the binary if needed
roxctl-development() {
   "$(roxctl-development-cmd)" "$@"
}

# roxctl-development-cmd prints the path to roxctl built with GOTAGS='release'. It builds the binary if needed
roxctl-release-cmd() {
  if [[ ! -x "${tmp_roxctl}/roxctl-release" ]]; then
    _uname="$(luname)"
    mkdir -p "$tmp_roxctl"
    make -s "cli-${_uname}" GOTAGS='release' 2>&3
    mv "bin/${_uname}/roxctl" "${tmp_roxctl}/roxctl-release"
  fi
  echo "${tmp_roxctl}/roxctl-release"
}

# roxctl-release runs roxctl built with GOTAGS='release'. It builds the binary if needed
roxctl-release() {
  "$(roxctl-release-cmd)" "$@"
}

helm_template_central() {
  local out_dir="${1}"
  run helm template stackrox-central-services "$out_dir" \
    -n stackrox \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"
  assert_success
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
}

assert_helm_template_central_registry() {
  local out_dir="${1}"; shift;
  helm_template_central "$out_dir"
  assert_components_registry "$out_dir/rendered/stackrox-central-services/templates" "$@"
}

wait_10s_for() {
  local file="$1"; shift
  local args=("${@}")
  for _ in {1..10}; do
    if "${args[@]}" "$file"; then return 0; fi
    sleep 1
  done
  "${args[@]}" "$file"
}

assert_sensor_component() {
  local dir="$1"
  local regex="$2"

  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "sensor").image' "${dir}/sensor.yaml"
  assert_output --regexp "$regex"
}

assert_collector_component() {
   local dir="$1"
   local regex="$2"

   run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "collector").image' "${dir}/collector.yaml"
   assert_output --regexp "$regex"
}

assert_secured_cluster_component_registry() {
  local dir="$1"
  local registry_slug="$2"
  shift; shift;

  [[ ! -d "$dir" ]] && fail "ERROR: not a directory: '$dir'"
  (( $# < 1 )) && fail "ERROR: 0 components provided"

  for component in "${@}"; do
    regex="$(registry_regex "$registry_slug" "$component")"
    case $component in
      main)
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "sensor").image' "${dir}/sensor.yaml"
        assert_output --regexp "$regex"
        ;;
      collector)
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "collector").image' "${dir}/collector.yaml"
        assert_output --regexp "$regex"
        ;;
      collector-slim)
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "collector").image' "${dir}/collector.yaml"
        assert_output --regexp "$regex"
        ;;
      *)
        fail "ERROR: unknown component: '$component'"
        ;;
    esac
  done
}

assert_components_registry() {
  local dir="$1"
  local registry_slug="$2"
  shift; shift;

  # The expect-based tests may be slow and flaky, so let's add timeouts to this assertion
  wait_10s_for "$dir" "test" "-d" || fail "ERROR: not a directory: '$dir'"
  (( $# < 1 )) && fail "ERROR: 0 components provided"

  for component in "${@}"; do
    regex="$(registry_regex "$registry_slug" "$component")"
    case $component in
      main)
        wait_10s_for "${dir}/01-central-12-deployment.yaml" "test" "-f" || fail "ERROR: file missing: '${dir}/01-central-12-deployment.yaml'"
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "central").image' "${dir}/01-central-12-deployment.yaml"
        assert_output --regexp "$regex"
        ;;
      scanner)
        wait_10s_for "${dir}/02-scanner-06-deployment.yaml" "test" "-f" || fail "ERROR: file missing: '${dir}/02-scanner-06-deployment.yaml'"
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "${dir}/02-scanner-06-deployment.yaml"
        assert_output --regexp "$regex"
        ;;
      scanner-db)
        wait_10s_for "${dir}/02-scanner-06-deployment.yaml" "test" "-f" || fail "ERROR: file missing: '${dir}/02-scanner-06-deployment.yaml'"
        run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "${dir}/02-scanner-06-deployment.yaml"
        assert_output --regexp "$regex"
        ;;
      sensor)
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "sensor").image' "${dir}/sensor.yaml"
        assert_output --regexp "$regex"
        ;;
      *)
        fail "ERROR: unknown component: '$component'"
        ;;
    esac
  done
}

# TODO ROX-9153 replace with bats-file
assert_file_exist() {
  local -r file="$1"
  if [[ ! -e "$file" ]]; then
    fail "ERROR: file '$file' does not exist"
  fi
}

registry_regex() {
  local registry_slug="$1"
  local component="$2"
  local version='[0-9]+\.[0-9]+\.'

  case $registry_slug in
    docker.io)
      echo "docker\.io/stackrox/$component:$version"
      ;;
    stackrox.io)
      echo "stackrox\.io/$component:$version"
      ;;
    collector.stackrox.io)
      echo "collector.stackrox\.io/$component:$version"
      ;;
    registry.redhat.io)
      echo "registry\.redhat\.io/advanced-cluster-security/rhacs-$component-rhel8:$version"
      ;;
    example.com)
      echo "example\.com/$component:$version"
      ;;
    example2.com)
      echo "example2\.com/$component:$version"
      ;;
    *)
      fail "ERROR: unknown registry-slug: '$registry_slug'"
      ;;
  esac
}

# Central-generate

# run_image_defaults_registry_test runs `roxctl central generate` and asserts the image registries match the expected values.
# Parameters:
# $1 - path to roxctl binary
# $2 - orchestrator (k8s, openshift)
# $3 - registry-slug for expected main registry (see 'registry_regex()' for the list of currently supported registry-slugs)
# $4 - registry-slug for expected scanner and scanner-db registries (see 'registry_regex()' for the list of currently supported registry-slugs)
# $@ - open-ended list of other parameters that should be passed into 'roxctl central generate'
run_image_defaults_registry_test() {
  local roxctl_bin="$1"; shift;
  local orch="$1"; shift;
  local expected_main_registry="$1"; shift;
  local expected_scanner_registry="$1"; shift;
  local extra_params=("${@}")

  [[ -n "$out_dir" ]] || fail "out_dir is unset"

  run "$roxctl_bin" central generate "$orch" "${extra_params[@]}" pvc --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" "$expected_main_registry" 'main'
  assert_components_registry "$out_dir/scanner" "$expected_scanner_registry" 'scanner' 'scanner-db'
}

# run_no_rhacs_flag_test asserts that 'roxctl central generate' fails when presented with `--rhacs` parameter
run_no_rhacs_flag_test() {
  local roxctl_bin="$1"
  local orch="$2"

  run "$roxctl_bin" central generate --rhacs "$orch" pvc --output-dir "$(mktemp -d -u)"
  assert_failure
  assert_output --partial "unknown flag: --rhacs"
  run "$roxctl_bin" central generate "$orch" --rhacs pvc --output-dir "$(mktemp -d -u)"
  assert_failure
  assert_output --partial "unknown flag: --rhacs"
}

# run_invalid_flavor_value_test asserts that 'roxctl central generate' fails when presented invalid value of `--image-defaults` parameter
run_invalid_flavor_value_test() {
  local roxctl_bin="$1"; shift;
  local orch="$1"; shift;
  local extra_params=("${@}")

  run "$roxctl_bin" central generate "$orch" "${extra_params[@]}" pvc --output-dir "$(mktemp -d -u)"
  assert_failure
  assert_output --regexp "invalid arguments: '--image-defaults': unexpected value .*, allowed values are \[.*\]"
}

# run_with_debug_flag_test copies chart bundle content into a temporary folder, modifies it, and executes a given command with the debug flag
run_with_debug_flag_test() {
  # default debug path argument
  local chart_src_dir="$GOPATH/src/github.com/stackrox/stackrox/image"
  [[ -d "$chart_src_dir" ]] || skip "This test requires a chart template located on the file system"

  [[ -n "$chart_debug_dir" ]] || fail "chart_debug_dir is unset"

  cp -r "$chart_src_dir" "$chart_debug_dir"
  # creating a diff between original and custom chart template to verify that the custom chart is used instead of the default one
  touch "$chart_debug_dir/templates/helm/shared/templates/bats-test.yaml"

  run "$@" --debug --debug-path "$chart_debug_dir"
}

assert_debug_templates_exist() {
    local tpl_dir="${1}"
    assert_file_exist "$tpl_dir/bats-test.yaml"
}

has_deprecation_warning() {
  assert_line --regexp "WARN:[[:space:]]+'--rhacs' is deprecated, please use '--image-defaults=rhacs' instead"
}

flavor_warning_regexp="WARN:[[:space:]]+Default image registries have changed. Images will be taken from 'registry.redhat.io'. Specify '--image-defaults=stackrox.io' command line argument to use images from 'stackrox.io' registries."

has_default_flavor_warning() {
  assert_line --regexp "$flavor_warning_regexp"
}

has_no_default_flavor_warning() {
  refute_line --regexp "$flavor_warning_regexp"
}

has_flag_collision_warning() {
  assert_line --partial "flag '--rhacs' is deprecated and must not be used together with '--image-defaults'. Remove '--rhacs' flag and specify only '--image-defaults'"
}

bundle_unique_name() {
  echo "bats-cluster-$(date '+%s')"
}

generate_bundle() {
  installation_flavor="$1";shift
  run roxctl-development sensor generate "$installation_flavor" \
        --insecure-skip-tls-verify -e "$API_ENDPOINT" "$@" \
        --output-dir="$out_dir" \
        --timeout=10m \
        --continue-if-exists
}

delete_cluster() {
  local name="$1";shift
  run roxctl-development cluster delete --name "$name" \
    -e "$API_ENDPOINT"
  assert_success
}
