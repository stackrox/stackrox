#!/bin/bash

set -ex

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

OC_PROJECT="${OC_PROJECT:-stackrox}"
OC_NAMESPACE="${OC_NAMESPACE:-stackrox}"
OC_SA="${OC_SA:-central}"

oc login -u system:admin && oc project default

PRIVATE_REGISTRY=$(oc get route -n default | grep docker-registry | tr -s ' ' | cut -d' ' -f2)
echo "Private registry: $PRIVATE_REGISTRY"

oc create -f scc-central.yaml || true

oc project $OC_PROJECT

echo "Setting up service account..."
oc get sa "$OC_SA" || oc create serviceaccount "$OC_SA"

oc secrets link --for=pull $OC_SA stackrox

echo "Adding cluster roles to the service account..."
oc project default
oc adm policy add-scc-to-user central "system:serviceaccount:$OC_PROJECT:$OC_SA"


oc project "$OC_PROJECT"
oc policy add-role-to-user edit "system:serviceaccount:$OC_PROJECT:$OC_SA" -n "$OC_PROJECT"
oc policy add-role-to-user system:image-puller "system:serviceaccount:$OC_PROJECT:$OC_SA" -n "$OC_PROJECT"

########################################

oc project "$OC_PROJECT"
