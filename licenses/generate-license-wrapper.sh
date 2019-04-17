#!/usr/bin/env bash

set -euo pipefail

GENERATE_LICENSE_VERSION="v0.0.0-24-g21fa2a4f3e"

generate_license_path="gs://stackrox-licensing-tools/generate-license/${GENERATE_LICENSE_VERSION}/$(uname | tr 'A-Z' 'a-z')/generate-license"

generate_license_bin="/tmp/generate-license-${GENERATE_LICENSE_VERSION}"
if [[ ! -x "$generate_license_bin" ]]; then
    gsutil cp "$generate_license_path" "$generate_license_bin"
    chmod a+x "$generate_license_bin"
fi

profile="$1"
shift

dir="$(dirname $0)"

if ! "$generate_license_bin" -config "${dir}/config.yaml" -profile "$profile" -input "${dir}/templates/${profile}.json" "$@"; then
    echo >&2 'Generating a license key failed. If the error message mentions credentials or'
    echo >&2 'insufficient permissions, run the `licenses/setup-gcloud.sh` script and try again.'
    exit 1
fi
