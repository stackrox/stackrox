#!/usr/bin/env bash

set -euo pipefail

GENERATE_LICENSE_VERSION="v0.0.0-17-ga1bf9f7cf5"

generate_license_path="gs://stackrox-licensing-tools/generate-license/${GENERATE_LICENSE_VERSION}/$(uname | tr 'A-Z' 'a-z')/generate-license"

if [[ ! -x /tmp/generate-license ]]; then
    gsutil cp "$generate_license_path" /tmp/generate-license
    chmod a+x /tmp/generate-license
fi

dir="$(dirname "$0")"
/tmp/generate-license \
    -not-valid-after +6h \
    -config "${dir}/generate-config.yaml" \
    -input "${dir}/license-template.json"
