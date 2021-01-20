#!/usr/bin/env bash

set -euo pipefail

dir="$(dirname "$0")"

"${dir}/generate-license-wrapper.sh" upgrade -not-valid-after +6h "$@"
