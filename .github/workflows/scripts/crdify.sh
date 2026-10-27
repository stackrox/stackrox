#!/usr/bin/env bash
# Run crdify on copies of CRDs after stripping out known false positives.
# Usage:
#   crdify.sh <old-crd> <new-crd> [crdify flags]

set -euo pipefail

ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/../../.."
readonly ROOT_DIR

if [[ $# -lt 2 ]]; then
    echo "usage: $0 <old-crd> <new-crd> [crdify flags]" >&2
    exit 2
fi

old_crd="${1:?missing required argument 'old-crd'}"
new_crd="${2:?missing required argument 'new-crd'}"
shift
shift

if [[ ! -f "$old_crd" ]]; then
    echo "::error::CRD file not found: $old_crd" >&2
    exit 1
fi

if [[ ! -f "$new_crd" ]]; then
    echo "::error::CRD file not found: $new_crd" >&2
    exit 1
fi


# The addition of this particular validation rule, even though considered a breaking change by definition
# (https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api_changes.md#backward-compatibility-gotchas)
# is fine in practice, since the `excludeByKind` branch is being added in the same release cycle as the rule itself,
# meaning no production CRS could have existed that violated the validation.
filter_spec_exclusions_one_of() {
    local input="$1"
    local output="$2"

    local message='at most one of the fields in [deployment excludeByKind] may be set'
    local rule='(has(self.deployment)?1:0)+(has(self.excludeByKind)?1:0) <= 1'

    # Remove matching rule, and the validations list if empty afterwards.
    message="${message}" rule="${rule}" \
    go run -C "${ROOT_DIR}/operator/tools/yq" github.com/mikefarah/yq/v4 eval '
      del(
        .spec.versions[].schema.openAPIV3Schema.properties.spec.properties.exclusions.items."x-kubernetes-validations"[]
        | select(.message == strenv(message) and .rule == strenv(rule))
      )
      | del(
          .spec.versions[].schema.openAPIV3Schema.properties.spec.properties.exclusions.items."x-kubernetes-validations"
          | select(length == 0)
        )
    ' "$input" > "$output"
}


crd_basename="$(basename "${old_crd}")"
case "$crd_basename" in
config.stackrox.io_securitypolicies.yaml)
    filtered_dir="$(mktemp -d)"
    echo >&2 "Filtering known false positives from ${old_crd} and ${new_crd}"
    filter_spec_exclusions_one_of "${old_crd}" "${filtered_dir}/old-${crd_basename}"
    filter_spec_exclusions_one_of "${new_crd}" "${filtered_dir}/new-${crd_basename}"
    old_crd="${filtered_dir}/old-${crd_basename}"
    new_crd="${filtered_dir}/new-${crd_basename}"
    ;;
*)
    echo >&2 "No filtering rules for ${old_crd}, running crdify on original CRDs."
    ;;
esac

set -x
go run -C "${ROOT_DIR}/operator/tools/crdify" sigs.k8s.io/crdify --config=config.yaml "$@" \
    "file://${old_crd}" "file://${new_crd}"
