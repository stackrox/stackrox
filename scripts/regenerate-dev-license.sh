#!/usr/bin/env bash

set -euo pipefail

dir="$(dirname "${BASH_SOURCE[0]}")"

generate_license_wrapper="${dir}/../licenses/generate-license-wrapper.sh"

license_duration="$((30 * 24))h"

license_key="$("$generate_license_wrapper" dev -not-valid-after "+${license_duration}" -loglevel debug)"

echo "$license_key" >"${dir}/../deploy/common/dev-license.lic"

echo >&2 "Re-generated development license"
