#!/usr/bin/bash

source "$(dirname "$0")/common.sh"

image_spec_prefix=quay.io/rhacs-eng/stackrox-operator-index
old_version=3.72.0
operator_ns=operators
central_ns=acs

if [ $(kubectl get deploy -n olm -o json | jq '.items | length') == 3 ]; then
    log OLM is already installed
else
    log Installing OLM
    # Download operator-sdk binary here:
    # https://github.com/operator-framework/operator-sdk/releases/tag/v1.25.4
    operator-sdk olm install
fi

kubectl create ns ${operator_ns}

create_pull_secret "${operator_ns}" "quay.io"

cat <<EOF | kubectl apply -n ${operator_ns} -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cert-manager
spec:
  channel: stable
  name: cert-manager
  source: operatorhubio-catalog
  sourceNamespace: olm
  installPlanApproval: Automatic
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: rhacs
spec:
  displayName: Advanced Cluster Security
  grpcPodConfig:
    securityContextConfig: restricted
  image: ${image_spec_prefix}:v${old_version}
  publisher: Red Hat
  secrets:
  - operator-pull-secret
  sourceType: grpc
  updateStrategy:
    registryPoll:
      interval: 60m
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: rhacs
spec:
  channel: latest
  name: rhacs-operator
  source: rhacs
  sourceNamespace: ${operator_ns}
  installPlanApproval: Automatic  # Choosing automatic to emulate how most customers configure subscriptions (afiact)
  config:
    env:
    # use a test value for NO_PROXY. This will not have any impact
    # on the services at runtime, but we can test if it gets piped
    # through correctly.
    - name: NO_PROXY
      value: "127.1.2.3/8"
EOF

if [ $? != 0 ];then
    log Failed to apply operators manifest
    exit 1
fi

nurse_deployment_until_available "${operator_ns}" "${old_version}"

kubectl create ns ${central_ns}

create_pull_secret "${central_ns}" "quay.io"

cat <<EOF | kubectl apply -n ${central_ns} -f -
---
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
spec:
  imagePullSecrets:
  - name: operator-pull-secret
  central:
    adminPasswordSecret:
      name: admin-pass
---
apiVersion: v1
kind: Secret
metadata:
  name: admin-pass
data:
  # letmein
  password: bGV0bWVpbg==
EOF

function test() {
  kubectl -n ${central_ns} wait --for=condition=Deployed --timeout=5m central/stackrox-central-services
  kubectl -n ${central_ns} wait --for=condition=Available --timeout=5m deploy/central

  kubectl -n ${central_ns} port-forward svc/central 44444:443 &
  sleep 2 # allow for port forward to get set up
  test_result=$(curl -k -u 'admin:letmein' https://localhost:44444/api/docs/swagger -s | jq -r .info.title)
  kill %1

  if [ "${test_result}" != "API Reference" ]; then
      log Failed to query API
      exit 1
  fi
}

test

new_version=3.74.0-13-gc76ca946d4 # perhaps `make tag` would make this dyanmic?

kubectl -n ${operator_ns} patch catalogsource/rhacs --type merge -p '{"spec":{"image":"'"${image_spec_prefix}"':v'"${new_version}"'"}}'

nurse_olm_upgrade "${operator_ns}" "${old_version}" "${new_version}"

"${KUTTL}" assert --timeout 300 --namespace ${central_ns} /dev/stdin <<-END
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
status:
  productVersion:  $(echo ${new_version} | sed s/\.0-/.x-/)
END

test

# Remove operator and OLM subscription
kubectl -n ${operator_ns} delete csv rhacs-operator.v${new_version}
kubectl -n ${operator_ns} delete subscription rhacs

# Rollback
kubectl -n ${operator_ns} patch catalogsource/rhacs --type merge -p '{"spec":{"image":"'"${image_spec_prefix}"':v'"${old_version}"'"}}'

cat <<EOF | kubectl apply -n ${operator_ns} -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: rhacs
spec:
  channel: latest
  name: rhacs-operator
  source: rhacs
  sourceNamespace: ${operator_ns}
  installPlanApproval: Automatic
  config:
    env:
    # use a test value for NO_PROXY. This will not have any impact
    # on the services at runtime, but we can test if it gets piped
    # through correctly.
    - name: NO_PROXY
      value: "127.1.2.3/8"
EOF

nurse_deployment_until_available "${operator_ns}" "${old_version}"

"${KUTTL}" assert --timeout 300 --namespace ${central_ns} /dev/stdin <<-END
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
status:
  productVersion:  ${old_version}
END
