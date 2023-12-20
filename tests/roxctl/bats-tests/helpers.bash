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
test_data="$BATS_TEST_DIRNAME/../test-data"

any_version='[0-9]+\.[0-9]+\.'

# roxctl-development-cmd prints the path to roxctl built with GOTAGS=''. It builds the binary if needed
roxctl-development-cmd() {
  if [[ ! -x "${tmp_roxctl}/roxctl-dev" ]]; then
    _uname="$(luname)"
    mkdir -p "$tmp_roxctl"
    make -s "cli-${_uname}" GOTAGS='' 2>&3
    mv "bin/${_uname}_amd64/roxctl" "${tmp_roxctl}/roxctl-dev"
  fi
  echo "${tmp_roxctl}/roxctl-dev"
}

# roxctl-development runs roxctl built with GOTAGS=''. It builds the binary if needed
roxctl-development() {
   "$(roxctl-development-cmd)" "$@"
}

# roxctl-release-cmd prints the path to roxctl built with GOTAGS='release'. It builds the binary if needed
roxctl-release-cmd() {
  if [[ ! -x "${tmp_roxctl}/roxctl-release" ]]; then
    _uname="$(luname)"
    mkdir -p "$tmp_roxctl"
    make -s "cli-${_uname}" GOTAGS='release' 2>&3
    mv "bin/${_uname}_amd64/roxctl" "${tmp_roxctl}/roxctl-release"
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
    --output-dir="$out_dir/rendered" \
    --set central.persistence.none=true
  assert_success
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/01-central-13-deployment.yaml"
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
}

# helm_template shall be used instead of `helm template --output-dir` to avoid potential overwriting of resulting manifests
helm_template() {
  local chart_dir="$1"; shift
  local out_dir="$1"; shift
  local helm_args=("${@}")
  # The following command overwrites some manifests in the out_dir:
  # helm template stackrox-secured-cluster-services "$out_dir" "${helm_args[@]}" --output-dir="$out_dir/rendered"
  # So we are printing to stdout and splitting output into files (inspired by: https://github.com/helm/helm/issues/4680)
  mkdir -p "$out_dir"
  helm template stackrox-secured-cluster-services "$chart_dir" "${helm_args[@]}" > "$out_dir/in.yaml"

  awk \
    -vout="$out_dir" \
    -F": " '$0~/^# Source: /{
      file=out"/"$2;
      system ("mkdir -p $(dirname "file"); touch "file)
      print "---" >> file
    } $0!~/^#/ && $0!="---"{
      print $0 >> file
    }' "$out_dir/in.yaml"
}

helm_template_secured_cluster() {
  local in_dir="${1}"
  local out_dir="${2}"
  local cluster_name="${3}"
  shift; shift; shift
  [[ -n "$cluster_name" ]] || fail "helm_template_secured_cluster: missing cluster_name"
  local extra_helm_args=("${@}")

  # Simulate: roxctl central init-bundles fetch-ca --output "$in_dir/ca-config.yaml"
  cp "$test_data/helm-output-secured-cluster-services/ca-config.yaml" "$in_dir/ca-config.yaml"
  yaml_valid "$in_dir/ca-config.yaml"

  # Simulate: roxctl central init-bundles generate "bundle-${cluster_name}" --output "$in_dir/cluster-init-bundle.yaml"
  cp "$test_data/helm-output-secured-cluster-services/cluster-init-bundle.yaml" "$in_dir/cluster-init-bundle.yaml"
  yaml_valid "$in_dir/cluster-init-bundle.yaml"

  helm_args=(
    --debug
    -n stackrox
    -f "$in_dir/cluster-init-bundle.yaml"
    -f "$in_dir/ca-config.yaml"
    --set "clusterName=${cluster_name}"
    --set "imagePullSecrets.allowNone=true"
    "${extra_helm_args[@]}"
  )

  rm -rf "$out_dir"
  run helm_template "$in_dir" "$out_dir" "${helm_args[@]}"
  assert_success
  assert_file_exist "$out_dir/stackrox-secured-cluster-services/templates/collector.yaml"
  assert_file_exist "$out_dir/stackrox-secured-cluster-services/templates/admission-controller.yaml"
  assert_file_exist "$out_dir/stackrox-secured-cluster-services/templates/sensor.yaml"
}

assert_helm_template_central_registry() {
  local out_dir="${1}"; shift;
  local registry_slug="${1}"; shift;
  local version_regex="${1}"; shift;
  helm_template_central "$out_dir"
  assert_components_registry "$out_dir/rendered/stackrox-central-services/templates" "$registry_slug" "$version_regex" "$@"
}

wait_20s_for() {
  local file="$1"; shift
  local args=("${@}")
  for _ in {1..20}; do
    if "${args[@]}" "$file"; then return 0; fi
    sleep 1
  done
  "${args[@]}" "$file"
}


assert_registry_version_file() {
  local file="$1"
  local doc_index="$2"
  local component="$3"
  local regex="$4"
  wait_20s_for "$file" "test" "-f" || fail "ERROR: file missing: '$file'"
  run yq e "select(documentIndex == $doc_index) | .spec.template.spec.containers[] | select(.name == \"${component}\").image" "${file}"
  assert_output --regexp "$regex"
}

assert_bundle_registry() {
  local dir="$1"
  local component="$2"
  local regex="$3"
  assert_registry_version_file "${dir}/${component}.yaml" 0 "$component" "$regex"
}

component_image() {
  case "$1" in
      admission-controller|sensor)
        echo "main"
        ;;
      *)
        echo "$1"
        ;;
    esac
}

assert_components_registry() {
  local dir="$1"
  local registry_slug="$2"
  local version_regex="$3"
  shift; shift; shift;

  # The expect-based tests may be slow and flaky, so let's add timeouts to this assertion
  wait_20s_for "$dir" "test" "-d" || fail "ERROR: not a directory: '$dir'"
  (( $# < 1 )) && fail "ERROR: 0 components provided"

  for component in "${@}"; do
    image="$(component_image "$component")"
    regex="$(image_reference_regex "$registry_slug" "$image" "$version_regex")"
    case $component in
      main)
        assert_registry_version_file "${dir}/01-central-13-deployment.yaml" 0 "central" "$regex"
        ;;
      central-db)
        assert_registry_version_file "${dir}/01-central-12-central-db.yaml" 0 "central-db" "$regex"
        ;;
      scanner)
        assert_registry_version_file "${dir}/02-scanner-06-deployment.yaml" 0 "scanner" "$regex"
        ;;
      scanner-db)
        assert_registry_version_file "${dir}/02-scanner-06-deployment.yaml" 1 "db" "$regex"
        ;;
      collector|collector-slim) # only when generated by helm output
        assert_registry_version_file "${dir}/stackrox-secured-cluster-services/templates/collector.yaml" 0 "collector" "$regex"
        ;;
      admission-controller) # only when generated by helm output
        assert_registry_version_file "${dir}/stackrox-secured-cluster-services/templates/admission-controller.yaml" 1 "admission-control" "$regex"
        ;;
      sensor) # only when generated by helm output
        assert_registry_version_file "${dir}/stackrox-secured-cluster-services/templates/sensor.yaml" 2 "sensor" "$regex"
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

assert_file_not_exist() {
  local -r file="$1"
  if [[ -e "$file" ]]; then
    fail "ERROR: file '$file' exists"
  fi
}

image_reference_regex() {
  local registry_slug="$1"
  local component="$2"
  local version="${3:-$any_version}"

  case "$registry_slug" in
    quay.io/rhacs-eng)
      echo "quay\.io/rhacs-eng/$component:$version"
      ;;
    quay.io/stackrox-io)
      echo "quay\.io/stackrox-io/$component:$version"
      ;;
    stackrox.io)
      if [[ "$component" == "collector" ]]; then
        echo "collector.stackrox\.io/$component:$version"
      else
        echo "stackrox\.io/$component:$version"
      fi
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
# $3 - registry-slug for expected main registry (see 'image_reference_regex()' for the list of currently supported registry-slugs)
# $4 - registry-slug for expected scanner and scanner-db registries (see 'image_reference_regex()' for the list of currently supported registry-slugs)
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
  assert_components_registry "$out_dir/central" "$expected_main_registry" "$any_version" 'main'
  assert_components_registry "$out_dir/scanner" "$expected_scanner_registry" "$any_version" 'scanner' 'scanner-db'
  assert_components_registry "$out_dir/central" "$expected_main_registry" "$any_version" 'central-db'
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
  assert_output --regexp "invalid command option: '--image-defaults': unexpected value .*, allowed values are \[.*\]"
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

has_flag_collision_warning() {
  assert_line --partial "flag '--rhacs' is deprecated and must not be used together with '--image-defaults'. Remove '--rhacs' flag and specify only '--image-defaults'"
}

roxctl_authenticated() {
  roxctl-development --insecure-skip-tls-verify -e "$API_ENDPOINT" -p "$ROX_PASSWORD" "$@"
}

yaml_valid() {
  assert_file_exist "$1"
  run yq e "$1"
  assert_success
}

generate_bundle() {
  installation_flavor="$1";shift
  run roxctl_authenticated sensor generate "$installation_flavor" \
        --output-dir="$out_dir" \
        --timeout=10m \
        --continue-if-exists \
        "$@"
}

delete_cluster() {
  local name="$1";shift
  run roxctl_authenticated cluster delete --name "$name" --timeout=1m
  assert_success
}
