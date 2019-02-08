#!/usr/bin/env bash

set -e

# check if port 8000 is already being forwarded
if [[ "$(lsof -n -i tcp:8000 | wc -l)" -gt 0 ]]; then
  echo "WARNING: Port 8000 is bound. Please first kill the process port-forwarding right now.";
  echo "You can either do 'kill <pid>' or 'killall kubectl' (to kill all running kubectl processes)";
  lsof -n -i tcp:8000;
  exit 1
fi

echo 'Connecting...'
central_pod="$(kubectl get pod -n stackrox --selector 'app=central' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
[[ -n  "${central_pod}" ]] || {
  echo "Couldn't find the Central pod! Have you connected and deployed?";
  exit 1;
}

kubectl port-forward -n stackrox "${central_pod}" 8000:443 & >/dev/null 2>&1
