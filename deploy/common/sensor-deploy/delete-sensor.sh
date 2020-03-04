#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl -n stackrox delete cm,secret,sa,svc,validatingwebhookconfigurations,ds,deploy,netpol,psp,clusterrole,clusterrolebinding,role,rolebinding -l auto-upgrade.stackrox.io/component=sensor --wait

if ! kubectl get -n stackrox deploy/central; then
    kubectl delete -n stackrox secret stackrox

fi
