#!/usr/bin/env bash

gcloud container clusters create \
    --machine-type n1-standard-2 \
    --num-nodes 3 \
    --create-subnetwork range=/19 \
    --enable-ip-alias \
    --enable-network-policy \
    --image-type UBUNTU \
    "prevent-ci-${CIRCLE_BUILD_NUM}"
