#!/usr/bin/env bash
set -eo pipefail

SCRIPT="$(python -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"
source "$(dirname "$SCRIPT")/common-vars.sh"

echo "Reinstall secured cluster services"
"$(dirname "$SCRIPT")"/uninstall-secured-cluster-services.sh
"$(dirname "$SCRIPT")"/install-secured-cluster-services.sh
