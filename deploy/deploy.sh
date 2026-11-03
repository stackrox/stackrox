#!/usr/bin/env bash
set -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd)"

echo "Using roxie to deploy StackRox. 🎉"
echo

if [[ "${SKIP_ROXIE_DEPLOY_BANNER:-false}" != "true" ]]; then
    cat <<EOF
----------------------------------------------------------------------------------------
This is a limited shell wrapper leveraging roxie under the hood.
In particular, you won't be able to configure feature flags by setting environment
variables. Instead, you can pass roxie command line arguments to this script,
for example:

  ${ROOT}/deploy/deploy.sh --features +ROX_I_WANT_THIS,-ROX_AND_NOT_THIS

roxie will deploy StackRox and then spawn a sub-shell for you to interact with central.
If the shell is closed accidentally, you can use

  ${ROOT}/deploy/deploy.sh shell

for re-entering the roxie sub-shell.

For tearing down the deployment, invoke:
  ${ROOT}/scripts/teardown.sh

You can find quick start instructions for roxie in:

    ${ROOT}/deploy/README.md.

If you do not want to see this banner again, set

  SKIP_ROXIE_DEPLOY_BANNER=true

----------------------------------------------------------------------------------------
EOF
fi

if [[ "${1:-}" == "shell" ]]; then
    "${ROOT}/scripts/roxie.sh" shell
else
    "${ROOT}/scripts/roxie.sh" --config "${ROOT}/deploy/roxie-config.yaml" deploy "$@"
fi
