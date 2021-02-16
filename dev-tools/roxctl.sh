#!/usr/bin/env bash
set -eo pipefail

# Usage: ./roxctl.sh <args>
# Small development wrapper around roxctl.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

roxctl_bin="$DIR/../bin/linux/roxctl"
if [[ "$(uname)" == "Darwin"* ]]; then
  roxctl_bin="$DIR/../bin/darwin/roxctl"
fi

"$roxctl_bin" -e "https://localhost:8000" -p "$(cat "$DIR/../deploy/k8s/central-deploy/password")" --insecure-skip-tls-verify "$@"
