#!/usr/bin/env bash
# Copy a SecurityPolicy CRD and drop one x-kubernetes-validations entry.
#
# The CEL rule stays in the CRD that is installed. crdify 0.6.0 has no
# validator for x-kubernetes-validations and reports that addition as an
# unhandled error, so Check CRD Compatibility compares this filtered copy.
#
# Usage:
#   strip-securitypolicy-atmostoneof.sh <input-crd> <output-crd>
#
# Requires: yq.

set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "usage: $0 <input-crd> <output-crd>" >&2
    exit 2
fi

input="${1:?missing required argument 'input-crd'}"
output="${2:?missing required argument 'output-crd'}"

if [[ ! -f "$input" ]]; then
    echo "::error::CRD file not found: $input" >&2
    exit 1
fi

# Match the generated rule, not every x-kubernetes-validations entry.
message='at most one of the fields in [deployment excludeByKind] may be set'
rule='(has(self.deployment)?1:0)+(has(self.excludeByKind)?1:0) <= 1'
export message rule

yq eval '
  del(
    .spec.versions[].schema.openAPIV3Schema.properties.spec.properties.exclusions.items."x-kubernetes-validations"[]
    | select(.message == strenv(message) and .rule == strenv(rule))
  )
  | del(
      .spec.versions[].schema.openAPIV3Schema.properties.spec.properties.exclusions.items."x-kubernetes-validations"
      | select(length == 0)
    )
' "$input" > "$output"
