#!/usr/bin/env bash

NAMESPACE=${1:-stackrox}
LB_IP=""

until [ -n "${LB_IP}" ] && [ "${LB_IP}" != "null" ]; do
    LB_IP=$(kubectl -n "${NAMESPACE}" get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname')
    sleep 1
done

echo "$LB_IP"
