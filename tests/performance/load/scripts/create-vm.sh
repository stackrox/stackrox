#!/usr/bin/env bash
set -eou pipefail

if ! command -v gcloud &> /dev/null; then
   echo "gcloud is missing. Install it"
   exit 1
fi

if [ "$#" -lt 3 ]; then
   echo "Usage: $0 <vm_name> <artifacts_dir> <rox_admin_passwword> [project] [roxctl_bin]"
fi

vm_name=$1
artifacts_dir=$2
rox_admin_password=$3
project="${4:-acs-team-sandbox}"
roxctl_bin="${5:-/usr/bin/roxctl}"
gcloud_config="${6:-${HOME}/.config/gcloud}"

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

echo "gcloud compute ssh --zone \"$zone\" "$vm_name" --project \""$project"\""
echo "$rox_admin_password" > /tmp/rox_admin_password.txt

gcloud compute scp "$gcloud_config" "$vm_name":~/gcloud --recurse --project "$project" > /dev/null
gcloud compute scp "$artifacts_dir" "$vm_name":~/artifacts --recurse --project "$project" > /dev/null
gcloud compute scp "/tmp/rox_admin_password.txt" "$vm_name":~/rox_admin_password.txt --project "$project" > /dev/null
gcloud compute scp "${DIR}/install-dependencies.sh" "$vm_name":~/install-dependencies.sh --project "$project" > /dev/null
gcloud compute scp "${DIR}/env.sh" "$vm_name":~/env.sh --project "$project" > /dev/null
gcloud compute scp "${DIR}/run.sh" "$vm_name":~/run.sh --project "$project" > /dev/null
gcloud compute scp "${DIR}/monitor-top-pod.sh" "$vm_name":~/monitor-top-pod.sh --project "$project" > /dev/null
gcloud compute scp "${DIR}/get-stackrox-info.sh" "$vm_name":~/get-stackrox-info.sh --project "$project" > /dev/null
gcloud compute scp "$roxctl_bin" "$vm_name":~/roxctl --recurse --project "$project" > /dev/null

echo "gcloud compute ssh --zone \"$zone\" "$vm_name" --project \""$project"\""
