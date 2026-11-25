#!/usr/bin/env bash
# Enable encryption of notifier secrets for a locally deployed (operator-managed) Central.
#
# This does two things:
# 1. Create the central-encryption-key-chain Secret holding the encryption key chain.
# 2. Enable notifier secrets encryption on the Central custom resource. The operator
#    then sets ROX_ENC_NOTIFIER_CREDS=true and mounts the Secret into the Central
#    deployment (see spec.central.notifierSecretsEncryption in the Central CRD).
set -eo pipefail

key_string=$(echo "key-string-1234" | base64)

echo "Creating (or updating) the central-encryption-key-chain Secret with a test key chain"
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

echo "Enabling notifierSecretsEncryption on the Central custom resource"
echo "The operator will restart Central with ROX_ENC_NOTIFIER_CREDS=true and the key chain mounted"
kubectl -n stackrox patch central stackrox-central-services --type=merge \
  -p '{"spec":{"central":{"notifierSecretsEncryption":{"enabled":true}}}}'

echo "Done. Notifier secrets stored in the Central DB are now encrypted with the test key chain."
