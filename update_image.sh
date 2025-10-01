#!/usr/bin/bash

if [ -z "$1" ]; then
    echo "Error: No image tag provided"
    exit 1
fi

image_tag="$1"
kubectl -n stackrox set image deploy/central "*=quay.io/rhacs-eng/main:$image_tag"
kubectl -n stackrox set image deploy/admission-control "*=quay.io/rhacs-eng/main:$image_tag"
kubectl -n stackrox set image daemonset/collector "compliance=quay.io/rhacs-eng/main:$image_tag"
kubectl -n stackrox set image deploy/sensor "*=quay.io/rhacs-eng/main:$image_tag"
kubectl -n stackrox set image deploy/central "*=quay.io/rhacs-eng/main:$image_tag"

kubectl -n stackrox set env deploy/sensor LOGLEVEL=debug ROX_VIRTUAL_MACHINES=true
kubectl -n stackrox set env deploy/central LOGLEVEL=debug ROX_VIRTUAL_MACHINES=true
kubectl -n stackrox set env daemonset/collector LOGLEVEL=info ROX_VIRTUAL_MACHINES=true
