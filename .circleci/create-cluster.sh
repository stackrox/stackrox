#!/usr/bin/env bash

### Network Sizing ###
# The overall subnetwork ("--create-subnetwork") is used for nodes.
# The "cluster" secondary range is for pods ("--cluster-ipv4-cidr").
# The "services" secondary range is for ClusterIP services ("--services-ipv4-cidr").
# See https://cloud.google.com/kubernetes-engine/docs/how-to/alias-ips#cluster_sizing.

create-cluster() {
  REGION=us-central1
  NUM_NODES="${NUM_NODES:-4}"
  GCP_IMAGE_TYPE="${GCP_IMAGE_TYPE:-UBUNTU}"

  # this function does not work in strict -e mode
  set +euo pipefail

  echo "Creating ${NUM_NODES} node cluster with image type \"${GCP_IMAGE_TYPE}\""

  zones=$(gcloud compute zones list --filter="region=$REGION" | grep UP | cut -f1 -d' ')
  success=0
  for zone in $zones; do
      echo "Trying zone $zone"
      gcloud config set compute/zone "${zone}"
      timeout 420 gcloud container clusters create \
          --machine-type n1-standard-2 \
          --num-nodes "${NUM_NODES}" \
          --create-subnetwork range=/28 \
          --cluster-ipv4-cidr=/20 \
          --services-ipv4-cidr=/24 \
          --enable-ip-alias \
          --enable-network-policy \
          --image-type ${GCP_IMAGE_TYPE} \
          --tags="stackrox-ci,stackrox-ci-${CIRCLE_JOB}" \
          "prevent-ci-${CIRCLE_BUILD_NUM}"
      status="$?"
      if [[ "${status}" == 0 ]];
      then
          success=1
          break
      elif [[ "${status}" == 124 ]];
      then
          echo >&2 "gcloud command timed out. Trying another zone..."
      fi
      echo >&2 "Deleting the cluster"
      gcloud container clusters delete "prevent-ci-${CIRCLE_BUILD_NUM}" --async
  done

  if [[ "${success}" == "0" ]]; then
      echo "Cluster creation failed"
      return 1
  fi

  # Sleep to ensure that GKE has actually started to create the deployments/pods
  sleep 10

  GRACE_PERIOD=30
  CURRENT_GRACE_PERIOD=0
  while true; do
    NUMSTARTING=$(kubectl -n kube-system get pod -o json | jq '(.items[].status.containerStatuses // [])[].ready' | grep false | wc -l | awk '{print $1}')
    if [[ "${NUMSTARTING}" == "0" ]]; then
      if (( CURRENT_GRACE_PERIOD >= GRACE_PERIOD )); then
        break
      fi
      sleep 5

      CURRENT_GRACE_PERIOD=$((CURRENT_GRACE_PERIOD + 5))
      echo "Current grace period set to ${CURRENT_GRACE_PERIOD}".

      continue
    fi

    # Reset the grace period if we find a pod that is not started
    CURRENT_GRACE_PERIOD=0

    echo "Waiting for ${NUMSTARTING} kube-system containers to be initialized"
    sleep 10
  done
}
