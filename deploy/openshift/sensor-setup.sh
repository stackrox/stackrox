#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

OC_PROJECT="${OC_PROJECT:-stackrox}"
OC_NAMESPACE="${OC_NAMESPACE:-stackrox}"
OC_SA="${OC_SA:-sensor}"
OC_BENCHMARK_SA="${OC_BENCHMARK_SA:-benchmark}"

oc login -u system:admin && oc project default

oc create -f scc-sensor.yaml || true
oc create -f scc-benchmark.yaml || true

oc project "$OC_PROJECT"

oc create -f rbac.yaml || true

echo "Setting up service account..."
oc get sa "$OC_SA" || oc create serviceaccount "$OC_SA"
oc secrets link --for=pull $OC_SA stackrox

oc get sa "$OC_BENCHMARK_SA" || oc create serviceaccount "$OC_BENCHMARK_SA"
oc secrets link --for=pull $OC_BENCHMARK_SA stackrox

echo "Adding cluster roles to the service account..."
oc login -u system:admin && oc project default
oc adm policy add-scc-to-user sensor "system:serviceaccount:$OC_PROJECT:$OC_SA"
oc adm policy add-scc-to-user benchmark "system:serviceaccount:$OC_PROJECT:$OC_BENCHMARK_SA"

oc project "$OC_PROJECT"
oc policy add-role-to-user edit "system:serviceaccount:$OC_PROJECT:$OC_SA" -n "$OC_PROJECT"
oc policy add-role-to-user system:image-puller "system:serviceaccount:$OC_PROJECT:$OC_BENCHMARK_SA" -n "$OC_PROJECT"

oc policy add-role-to-user system:image-puller "system:serviceaccount:$OC_PROJECT:$OC_BENCHMARK_SA" -n "$OC_PROJECT"

oc adm policy add-cluster-role-to-user monitor-deployments "system:serviceaccount:$OC_PROJECT:$OC_SA" -n "$OC_PROJECT"
oc adm policy add-cluster-role-to-user enforce-policies "system:serviceaccount:$OC_PROJECT:$OC_SA" -n "$OC_PROJECT"
oc adm policy add-role-to-user launch-benchmarks "system:serviceaccount:$OC_PROJECT:$OC_SA" -n "$OC_PROJECT"

oc project "$OC_PROJECT"
