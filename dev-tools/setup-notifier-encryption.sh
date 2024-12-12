#!/usr/bin/env bash
set -eo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd "$DIR"/..

key_string=$(echo "key-string-1234" | base64)
echo "
    apiVersion: v1
    stringData:
      key-chain.yaml: |
        keyMap:
          0: $key_string
        activeKeyId: 0
    kind: Secret
    metadata:
      name: central-encryption-key-chain
      namespace: stackrox
    type: Opaque
" | kubectl -n stackrox apply -f -

kubectl -n stackrox set env deployment/central ROX_ENC_NOTIFIER_CREDS=true

make -C "$DIR/../" cli-linux

"$DIR"/debug-helm-chart.sh -n stackrox upgrade stackrox-central-services --set central.notifierSecretsEncryption.enabled=true --reuse-values "$DIR"/../stackrox-central-services-chart
