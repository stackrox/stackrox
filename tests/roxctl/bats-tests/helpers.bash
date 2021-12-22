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

helm-template-central() {
  local out_dir="${1}"
  run helm template stackrox-central-services "$out_dir" \
    -n stackrox \
    --set imagePullSecrets.allowNone=true \
    --output-dir="$out_dir/rendered"
  assert_success
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/01-central-12-deployment.yaml"
  assert_output --partial "wrote $out_dir/rendered/stackrox-central-services/templates/02-scanner-06-deployment.yaml"
}

### Assertions

assert_central_image_matches() {
  run yq e '.spec.template.spec.containers[] | select(.name == "central").image' "$1"
  assert_output --regexp "$2"
}

assert_scanner_image_matches() {
  run yq e 'select(documentIndex == 0) | .spec.template.spec.containers[] | select(.name == "scanner").image' "$1"
  assert_output --regexp "$2"
}

assert_scanner_db_image_matches() {
  run yq e 'select(documentIndex == 1) | .spec.template.spec.containers[] | select(.name == "db").image' "$1"
  assert_output --regexp "$2"
}
