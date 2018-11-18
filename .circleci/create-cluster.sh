#!/usr/bin/env bash

### Network Sizing ###
# The overall subnetwork ("--create-subnetwork") is used for nodes.
# The "cluster" secondary range is for pods ("--cluster-ipv4-cidr").
# The "services" secondary range is for ClusterIP services ("--services-ipv4-cidr").
# See https://cloud.google.com/kubernetes-engine/docs/how-to/alias-ips#cluster_sizing.

gcloud container clusters create \
    --machine-type n1-standard-2 \
    --num-nodes 4 \
    --create-subnetwork range=/28 \
    --cluster-ipv4-cidr=/20 \
    --services-ipv4-cidr=/24 \
    --enable-ip-alias \
    --enable-network-policy \
    --image-type UBUNTU \
    --tags="stackrox-ci,stackrox-ci-${CIRCLE_JOB}" \
    "prevent-ci-${CIRCLE_BUILD_NUM}"
