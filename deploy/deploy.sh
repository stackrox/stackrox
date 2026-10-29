#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd)"

if [[ "${SKIP_ROXIE_DEPLOY_BANNER:-false}" != "true" ]]; then
    cat <<EOF
----------------------------------------------------------------------------------------
Deploying StackRox with roxie. 🎉

You can pass roxie command line arguments to this script, for example:

  ${ROOT}/deploy/deploy.sh --features +ROX_I_WANT_THIS,-ROX_AND_NOT_THIS

roxie will deploy StackRox and then spawn a sub-shell for you to interact with central.
If the shell is closed accidentally, you can use

  ${ROOT}/deploy/deploy.sh shell

for re-entering the roxie sub-shell.

For tearing down the deployment, invoke:
  ${ROOT}/scripts/teardown.sh

For more information, see:
  ${DIR}/README.md

To hide this banner, set SKIP_ROXIE_DEPLOY_BANNER=true
----------------------------------------------------------------------------------------
EOF
fi

if [[ "${1:-}" == "shell" ]]; then
    "${ROOT}/scripts/roxie.sh" shell
else
    "${ROOT}/scripts/roxie.sh" --config "${ROOT}/deploy/roxie-config.yaml" deploy "$@"
fi
