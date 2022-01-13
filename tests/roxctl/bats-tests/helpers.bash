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

# roxctl-development runs roxctl built with GOTAGS=''. It builds the binary if needed
roxctl-development() {
  if [[ ! -x "${tmp_roxctl}/roxctl-dev" ]]; then
    _uname="$(luname)"
    mkdir -p "$tmp_roxctl"
    GOTAGS='' make "cli-${_uname}"
    mv "bin/${_uname}/roxctl" "${tmp_roxctl}/roxctl-dev"
  fi
  "${tmp_roxctl}/roxctl-dev" "$@"
}

# roxctl-release runs roxctl built with GOTAGS='release'. It builds the binary if needed
roxctl-release() {
  if [[ ! -x "${tmp_roxctl}/roxctl-release" ]]; then
    _uname="$(luname)"
    mkdir -p "$tmp_roxctl"
    GOTAGS='release' make "cli-${_uname}"
    mv "bin/${_uname}/roxctl" "${tmp_roxctl}/roxctl-release"
  fi
  "${tmp_roxctl}/roxctl-release" "$@"
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

assert_components_registry() {
  local dir="$1"
  local registry_slug="$2"
  shift; shift;

  [[ ! -d "$dir" ]] && fail "ERROR: not a directory: '$dir'"
  (( $# < 1 )) && fail "ERROR: 0 components provided"

  for component in "${@}"; do
    regex="$(registry_regex "$registry_slug" "$component")"
    case $component in
      main)
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "central").image' "${dir}/01-central-12-deployment.yaml"
        assert_output --regexp "$regex"
        ;;
      scanner)
        run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "${dir}/02-scanner-06-deployment.yaml"
        assert_output --regexp "$regex"
        ;;
      scanner-db)
        run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "${dir}/02-scanner-06-deployment.yaml"
        assert_output --regexp "$regex"
        ;;
      *)
        fail "ERROR: unknown component: '$component'"
        ;;
    esac
  done
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
    registry.redhat.io-short)
      echo "registry\.redhat\.io/rh-acs/$component:$version"
      ;;
    registry.redhat.io)
      echo "registry\.redhat\.io/advanced-cluster-security/rhacs-rhel8-$component:$version"
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

skip_unless_image_defaults() {
  bin="${1:-roxctl}"
  orch="${2:-k8s}"
  grep -- "--image-defaults" <("$bin" central generate "$orch" -h) || skip "because roxctl generate $orch does not support --image-defaults flag yet"
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

  if [[ " ${extra_params[*]} " =~ --image-defaults ]]; then
    skip_unless_image_defaults "$roxctl_bin" "$orch"
  fi
  run "$roxctl_bin" central generate "$orch" "${extra_params[@]}" pvc --output-dir "$out_dir"
  assert_success
  assert_components_registry "$out_dir/central" "$expected_main_registry" 'main'
  assert_components_registry "$out_dir/scanner" "$expected_scanner_registry" 'scanner' 'scanner-db'
}

# run_no_rhacs_flag_test asserts that 'roxctl central generate' fails when presented with `--rhacs` parameter
run_no_rhacs_flag_test() {
  local roxctl_bin="$1"
  local orch="$2"

  if [[ " ${extra_params[*]} " =~ --image-defaults ]]; then
    skip_unless_image_defaults "$roxctl_bin" "$orch"
  fi
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

  if [[ " ${extra_params[*]} " =~ --image-defaults ]]; then
    skip_unless_image_defaults "$roxctl_bin" "$orch"
  fi
  run "$roxctl_bin" central generate "$orch" "${extra_params[@]}" pvc --output-dir "$(mktemp -d -u)"
  assert_failure
  assert_output --regexp "invalid value of '--image-defaults': unexpected value .*, allowed values are \[.*\]"
}
