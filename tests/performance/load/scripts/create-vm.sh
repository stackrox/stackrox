#!/usr/bin/env bash
set -eou pipefail

if ! command -v gcloud &> /dev/null; then
   echo "gcloud is missing. Install it"
   exit 1
fi

if [ "$#" -lt 2 ]; then
   echo "Usage: $0 <vm_name> <artifacts_dir> [project] [roxctl_bin]"
fi

vm_name=$1
artifacts_dir=$2
project="${3:-acs-team-sandbox}"
roxctl_bin="${4:-/usr/bin/roxctl}"

zone="us-east1-b"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check if the VM instance exists
if gcloud compute instances describe "$vm_name" --zone="$zone" --quiet > /dev/null 2>&1; then
  echo "Instance $vm_name exists in zone $zone."
else
  gcloud compute instances create --zone "$zone" \
    --image-family ubuntu-2204-lts \
    --image-project ubuntu-os-cloud \
    --project "$project" \
    --machine-type e2-standard-2 \
    --boot-disk-size=30GB "$vm_name"
  sleep 60
fi

gcloud compute scp "$artifacts_dir" "$vm_name":~/artifacts --recurse --project "$project"
gcloud compute scp "${DIR}/install-dependencies.sh" "$vm_name":~/install-dependencies.sh --project "$project"
gcloud compute scp "$roxctl_bin" "$vm_name":~/roxctl --recurse --project "$project"

echo "gcloud compute ssh --zone \"$zone\" "$vm_name" --project \""$project"\""
