#!/usr/bin/env bash
set -eou pipefail

vm_name=$1
infra_name=$2
oc_bin=$3
env_version=$4

zone="us-east1-b"
repo_root="$(cd "$(dirname "$0")/../../../.." && pwd)"

source "$env_version"
commit_hash="$(echo "$IMAGE_MAIN_TAG" | sed 's/.*-g//')"

echo "Building roxctl from commit ${commit_hash}"
original_ref="$(git -C "$repo_root" rev-parse --abbrev-ref HEAD)"
git -C "$repo_root" checkout "$commit_hash"
make -C "$repo_root" roxctl_linux-amd64
git -C "$repo_root" checkout "$original_ref"
roxctl_bin="${repo_root}/bin/linux_amd64/roxctl"

# Check if the VM instance exists
if gcloud compute instances describe "$vm_name" --zone="$zone" --quiet > /dev/null 2>&1; then
  echo "Instance $vm_name exists in zone $zone."
else
  gcloud compute instances create --zone "$zone" --image-family ubuntu-2204-lts --image-project ubuntu-os-cloud --project acs-team-sandbox --machine-type e2-standard-2 --boot-disk-size=30GB "$vm_name"
  sleep 60
fi

gcloud compute scp /tmp/artifacts-"${infra_name}" "$vm_name":~/artifacts --recurse --project acs-team-sandbox
gcloud compute scp "$oc_bin" "$vm_name":~/oc --recurse --project acs-team-sandbox
gcloud compute scp "$roxctl_bin" "$vm_name":~/roxctl --recurse --project acs-team-sandbox
gcloud compute scp run-perf.sh "$vm_name":~/run-perf.sh --recurse --project acs-team-sandbox
gcloud compute scp install-dependencies.sh "$vm_name":~/install-dependencies.sh --recurse --project acs-team-sandbox
gcloud compute scp setup-env-var.sh "$vm_name":~/setup-env-var.sh --recurse --project acs-team-sandbox
gcloud compute scp run-perf-tests.sh "$vm_name":~/run-perf-tests.sh --recurse --project acs-team-sandbox
gcloud compute scp install-and-run.sh "$vm_name":~/install-and-run.sh --recurse --project acs-team-sandbox
gcloud compute scp "$env_version" "$vm_name":~/env.sh --recurse --project acs-team-sandbox
gcloud compute scp perf-tests.json "$vm_name":~/perf-tests.json --recurse --project acs-team-sandbox

echo "gcloud compute ssh --zone \"$zone\" "$vm_name" --project \"acs-team-sandbox\""
