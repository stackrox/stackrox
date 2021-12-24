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
    *)
      fail "ERROR: unknown registry-slug: '$registry_slug'"
      ;;
  esac
}

skip_unless_rhacs() {
  grep "\-\-rhacs" <(roxctl central generate -h) || skip "because roxctl generate does not support --rhacs flag yet"
}

skip_unless_image_defaults() {
  grep "\-\-image\-defaults" <(roxctl central generate -h) || skip "because roxctl generate does not support --image-defaults flag yet"
}
