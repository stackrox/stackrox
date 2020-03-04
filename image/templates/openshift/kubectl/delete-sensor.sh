#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc -n stackrox delete cm,secret,sa,svc,validatingwebhookconfigurations,ds,deploy,netpol,psp,clusterrole,clusterrolebinding,role,rolebinding -l auto-upgrade.stackrox.io/component=sensor --wait

if ! oc get -n stackrox deploy/central > /dev/null; then
    oc delete -n stackrox secret stackrox
fi
