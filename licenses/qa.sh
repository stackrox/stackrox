#!/usr/bin/env bash

set -euo pipefail

dir="$(dirname "$0")"

"${dir}/generate-license-wrapper.sh" qa "$@"
