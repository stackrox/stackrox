#!/usr/bin/env bash
set -eou pipefail

# Running these commands one by one is recommended instead of running this script.
# Check the UI after each command.
# This assumes that ACS has been deployed with ROX_EXTERNAL_IPS and ROX_NETWORK_GRAPH_EXTERNAL_IPS

# Cleanup from previous runs
kubectl delete ns qa
kubectl -n stackrox delete configmap collector-config

kubectl create ns qa

# Create the image pull secret
./create-secret-for-qa.sh

# Enable collection of external IPs by collector 
./enable-collector-external-ips.sh

#sleep 60

# Create a deployment that reaches out to google in the qa namespace
kubectl create -f external-destination-source.yml 

# Create a deployment that reaches out to 8.8.8.8
./create-deployment-with-ext-ip.sh 8.8.8.8 53 1

# Create a CIDR block that matches the above deployment
#./create-cidr-block.sh 8.8.8.0/24 testCIDR


# Create a deployment that reaches out to 1.1.1.1
./create-deployment-with-ext-ip.sh 1.1.1.1 53 2

# Get a network policy based on the network graph
# and apply it
./get-network-policy.sh > net-pol.yml
kubectl create -f net-pol.yml

# Create a deployment that reaches out to 2.2.2.2
./create-deployment-with-ext-ip.sh 2.2.2.2 53 3
