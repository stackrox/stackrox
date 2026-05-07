#!/usr/bin/env bash
set -eou pipefail

vm_name=$1
infra_name=$2
oc_bin=$3
env_version=$4

zone="us-east1-b"

# Check if the VM instance exists
if gcloud compute instances describe "$vm_name" --zone="$zone" --quiet > /dev/null 2>&1; then
  echo "Instance $vm_name exists in zone $zone."
else
  gcloud compute instances create --zone "$zone" --image-family ubuntu-2204-lts --image-project ubuntu-os-cloud --project acs-team-sandbox --machine-type e2-standard-2 --boot-disk-size=30GB "$vm_name"
  sleep 60
fi

gcloud compute scp /tmp/artifacts-"${infra_name}" "$vm_name":~/artifacts --recurse --project acs-team-sandbox
gcloud compute scp "$oc_bin" "$vm_name":~/oc --recurse --project acs-team-sandbox
gcloud compute scp run-perf.sh "$vm_name":~/run-perf.sh --recurse --project acs-team-sandbox
gcloud compute scp install-dependencies.sh "$vm_name":~/install-dependencies.sh --recurse --project acs-team-sandbox
gcloud compute scp setup-env-var.sh "$vm_name":~/setup-env-var.sh --recurse --project acs-team-sandbox
gcloud compute scp run-perf-tests.sh "$vm_name":~/run-perf-tests.sh --recurse --project acs-team-sandbox
gcloud compute scp install-and-run.sh "$vm_name":~/install-and-run.sh --recurse --project acs-team-sandbox
gcloud compute scp "$env_version" "$vm_name":~/env.sh --recurse --project acs-team-sandbox
gcloud compute scp perf-tests.json "$vm_name":~/perf-tests.json --recurse --project acs-team-sandbox

echo "gcloud compute ssh --zone \"$zone\" "$vm_name" --project \"acs-team-sandbox\""
