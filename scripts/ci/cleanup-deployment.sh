#!/usr/bin/env bash

namespace=${1:-stackrox}

echo "unexpected execution of cleanup scripts - SKIPPING cleanup"
echo "[skipped cmd:] kubectl -n ${namespace} get cm,deploy,ds,networkpolicy,pv,pvc,secret,svc,serviceaccount -o name | xargs kubectl -n ${namespace} delete --wait"
echo 'exit 1 to fail if this is executed'
exit 1
