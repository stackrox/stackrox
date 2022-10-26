#!/usr/bin/env bash

NAMESPACE="${NAMESPACE:-stackrox}"

kubectl delete namespace "${NAMESPACE}" || true

NAMESPACE_GONE=1
until [ $NAMESPACE_GONE -eq 0 ]
do
    NAMESPACE_GONE=$(kubectl get namespaces -o json | jq .items[].status.phase | grep -c "Terminating")
    echo -en "\rTerminating StackRox namespace....                                     "
    sleep 1
done
echo -e "\rDONE"
