#!/usr/bin/env bash
set -eou pipefail

export ARTIFACTS_DIR="${HOME}/artifacts"
export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

export HOST="https://localhost:8000"

password_file=${HOME}/rox_admin_password.txt
if [ -e "$password_file" ]; then 
  rox_admin_password="$(cat "$password_file")"
  export ROX_ADMIN_PASSWORD="$rox_admin_password"
else
  echo "$rox_admin_password does not exist. Cannot set password"
fi
