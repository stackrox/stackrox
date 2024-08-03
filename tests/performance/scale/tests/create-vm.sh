#!/usr/bin/env bash
set -eou pipefail

vm_name=$1
oc_bin=$2
infractl_bin=$3
env_version=$4

DIR="$(cd "$(dirname "$0")" && pwd)"

zone="us-east1-b"
project="acs-team-sandbox"

# Check if the VM instance exists
if gcloud compute instances describe "$vm_name" --zone="$zone" --quiet > /dev/null 2>&1; then
  echo "Instance $vm_name exists in zone $zone."
else
  gcloud compute instances create --zone "$zone" --image-family ubuntu-2204-lts --image-project ubuntu-os-cloud --project "$project" --machine-type e2-standard-2 --boot-disk-size=30GB "$vm_name"
  sleep 60
fi

#gcloud compute scp /tmp/artifacts-"${infra_name}" "$vm_name":~/artifacts --recurse --project "$project"
gcloud compute scp "$oc_bin" "$vm_name":~/oc --project "$project"
gcloud compute scp "$infractl_bin" "$vm_name":~/infractl --project "$project"
#gcloud compute scp "${DIR}"/run-perf.sh "$vm_name":~/run-perf.sh --project "$project"
gcloud compute scp "${DIR}"/run-test.sh "$vm_name":~/run-test.sh --project "$project"
gcloud compute scp "${DIR}"/install-dependencies.sh "$vm_name":~/install-dependencies.sh --project "$project"
gcloud compute scp "${DIR}"/setup-env-var.sh "$vm_name":~/setup-env-var.sh --project "$project"
#gcloud compute scp "${DIR}"/run-perf-tests.sh "$vm_name":~/run-perf-tests.sh --project "$project"
gcloud compute scp "${DIR}"/run-perf-tests-2.sh "$vm_name":~/run-perf-tests-2.sh --project "$project"
gcloud compute scp "${DIR}"/install-and-run.sh "$vm_name":~/install-and-run.sh --project "$project"
gcloud compute scp "${DIR}"/create-cluster.sh "$vm_name":~/create-cluster.sh --project "$project"
gcloud compute scp "${DIR}"/isolate-monitoring.sh "$vm_name":~/isolate-monitoring.sh --project "$project"
gcloud compute scp "$env_version" "$vm_name":~/env.sh --project "$project"
gcloud compute scp "${DIR}"/perf-tests.json "$vm_name":~/perf-tests.json --project "$project"

echo "gcloud compute ssh --zone \"$zone\" \"$vm_name\" --project \"$project\""
