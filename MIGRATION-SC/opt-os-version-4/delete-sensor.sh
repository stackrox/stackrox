#!/usr/bin/env bash

oc -n stackrox delete cm,secret,sa,svc,validatingwebhookconfigurations,ds,deploy,netpol,clusterrole,clusterrolebinding,role,rolebinding -l auto-upgrade.stackrox.io/component=sensor --wait

oc -n kube-system delete rolebinding -l auto-upgrade.stackrox.io/component=sensor --wait
oc -n openshift-monitoring delete prometheusrule,servicemonitor -l auto-upgrade.stackrox.io/component=sensor --wait

SUPPORTS_PSP=$(oc api-resources | grep "podsecuritypolicies" -c || true)

if [[ "${SUPPORTS_PSP}" -ne 0 ]]; then
    oc -n stackrox delete psp -l auto-upgrade.stackrox.io/component=sensor --wait
fi

oc delete scc -l auto-upgrade.stackrox.io/component=sensor --wait

if ! oc get -n stackrox deploy/central > /dev/null 2>&1; then
    oc delete -n stackrox secret stackrox
fi
