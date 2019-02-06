#! /bin/sh

LB_IP=""
until [ -n "${LB_IP}" ]; do
    LB_IP=$(kubectl -n stackrox get svc/central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    sleep 1
done

echo $LB_IP
