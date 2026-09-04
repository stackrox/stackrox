#!/usr/bin/env bash

kubectl -n stackrox delete cm,secret,sa,svc,validatingwebhookconfigurations,ds,deploy,netpol,clusterrole,clusterrolebinding,role,rolebinding -l auto-upgrade.stackrox.io/component=sensor --wait

SUPPORTS_PSP=$(kubectl api-resources | grep "podsecuritypolicies" -c || true)

if [[ "${SUPPORTS_PSP}" -ne 0 ]]; then
    kubectl -n stackrox delete psp -l auto-upgrade.stackrox.io/component=sensor --wait
fi

if ! kubectl get -n stackrox deploy/central 2>&1; then
    kubectl delete -n stackrox secret stackrox
fi
