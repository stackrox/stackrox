#!/usr/bin/env bash

load_maybe() {
  if [[ -f "${1}" ]]; then
    load "${1}"
  fi
}

load_maybe "/usr/lib/node_modules/bats-support/load.bash"
load_maybe "/usr/lib/node_modules/bats-assert/load.bash"
# These lines allow to run the tests locally provided that bats-helpers are installed in $HOME/bats-core
load_maybe "${HOME}/bats-core/bats-support/load.bash"
load_maybe "${HOME}/bats-core/bats-assert/load.bash"

# luname outputs uname in lowercase
luname() {
  uname | tr A-Z a-z
}

# roxctl-development runs roxctl built with GOTAGS=''. It builds the binary if needed
roxctl-development() {
  _uname="$(luname)"
  if [[ ! -x "bin/${_uname}/roxctl-dev" ]]; then
    GOTAGS='' make "cli-${_uname}"
    mv "bin/${_uname}/roxctl" "bin/${_uname}/roxctl-dev"
  fi
  "bin/${_uname}/roxctl-dev" "$@"
}

# roxctl-release runs roxctl built with GOTAGS='release'. It builds the binary if needed
roxctl-release() {
  _uname="$(luname)"
  if [[ ! -x "bin/${_uname}/roxctl-release" ]]; then
    GOTAGS='release' make "cli-${_uname}"
    mv "bin/${_uname}/roxctl" "bin/${_uname}/roxctl-release"
  fi
  "bin/${_uname}/roxctl-release" "$@"
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
