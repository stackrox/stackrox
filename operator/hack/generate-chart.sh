#!/usr/bin/env bash
set -euo pipefail

# TODO(ROX-33603): change to rhel9 when it's time
readonly downstream_image="registry.redhat.io/advanced-cluster-security/rhacs-rhel8-operator"
operator_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

function usage() {
    echo >&2 "Usage: $0 <opensource|development_build|rhacs>"
    echo >&2
    echo >&2 "Generates the operator helm chart in ${operator_dir}/dist/chart"
    echo >&2
    echo >&2 "Argument          | Branding |  Operator Image Repository"
    echo >&2 "------------------+----------+-------------------------------------------------------------"
    echo >&2 "opensource        | StackRox | quay.io/stackrox-io/stackrox-operator"
    echo >&2 "development_build | RHACS    | quay.io/rhacs-eng/stackrox-operator"
    echo >&2 "rhacs             | RHACS    | ${downstream_image}"
    echo >&2
    echo >&2 "Note: If you want to skip regenerating proto files, prepend the invocation with:"
    echo >&2 "ROX_OPERATOR_SKIP_PROTO_GENERATED_SRCS=true"
    exit 1
}

if [[ "${1:---help}" = "--help" ]]; then
    usage
fi

case "$1" in
opensource)
  ROX_PRODUCT_BRANDING=STACKROX_BRANDING make -C "${operator_dir}" chart
  ;;
development_build)
  ROX_PRODUCT_BRANDING=RHACS_BRANDING make -C "${operator_dir}" chart
  ;;
rhacs)
  ROX_PRODUCT_BRANDING=RHACS_BRANDING make -C "${operator_dir}" chart IMAGE_TAG_BASE="${downstream_image}"
  ;;
*)
  echo >&2 "$0: unrecognized argument: $1"
  echo >&2
  usage
esac
