#!/usr/bin/env bash
set -eou pipefail

vm_name=$1
infra_name=$2
oc_bin=$3

gcloud compute instances create --image-family ubuntu-2204-lts --image-project ubuntu-os-cloud --project acs-team-sandbox --machine-type e2-standard-2 --boot-disk-size=30GB "$vm_name"

sleep 60

gcloud compute scp /tmp/artifacts-"${infra_name}" "$vm_name":~/artifacts --recurse --project acs-team-sandbox
gcloud compute scp "$oc_bin" "$vm_name":~/oc --recurse --project acs-team-sandbox
gcloud compute scp run-perf.sh "$vm_name":~/run-perf.sh --recurse --project acs-team-sandbox


echo "gcloud compute ssh --zone \"us-east1-b\" "$vm_name" --project \"acs-team-sandbox\""
