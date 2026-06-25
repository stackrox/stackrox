#!/usr/bin/env bash
set -euo pipefail

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd)"

"${ROOT}/scripts/roxie.sh" teardown --single-namespace
