#! /bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

echo "Deploying Compliance Operator"
git clone git@github.com:openshift/compliance-operator.git
cd compliance-operator

oc create -f deploy/ns.yaml
for f in $(ls -1 deploy/crds/*crd.yaml); do oc apply -f $f -n openshift-compliance; done
oc apply -n openshift-compliance -f deploy/

echo "Deploying Scan setting bindings"
oc -n openshift-compliance create -R -f "${DIR}/ssb"
