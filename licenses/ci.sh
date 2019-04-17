#!/usr/bin/env bash

set -euo pipefail

dir="$(dirname "$0")"

"${dir}/generate-license-wrapper.sh" ci -not-valid-after +6h "$@"
