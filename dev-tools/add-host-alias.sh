#!/usr/bin/env bash

# Usage: ./add-host-alias.sh [deployment name] [alias] [ip | service DNS]
#
# Examples:
# To set an alias to an ip_or_dns address:
# ./add-host-alias.sh central my-super-domain.com 10.98.147.38
#
# To set an alias to an Kubernetes service IP use:
# ./add-host-alias.sh central my-super-domain.com central.stackrox

if [[ "$#" -ne 3 ]]; then
  echo "Expected 3 args, but found $#"
  echo "Usage: ./add-host-alias.sh [deployment name] [alias] [ip | service DNS]"
  exit 1
fi

deployment_name="$1"
alias="$2"
ip_or_dns="$3"
target_ip=""

if [[ $ip_or_dns =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  target_ip="$ip_or_dns"
else
  echo "No valid IP address was found, falling back to resolve Kubernetes service DNS"
  svc_name=$(echo "$ip_or_dns" | awk -F'.' '{NF--; print}')
  svc_namespace=$(echo "$ip_or_dns" | awk -F'.' '{print $NF}')
  target_ip=$(kubectl -n "$svc_namespace" get svc "$svc_name" -o template --template={{.spec.clusterIP}})
  if [[ -z "$target_ip" ]]; then
    echo "Error: could not resolve '$ip_or_dns'"
    exit 1
  fi
  echo "Resolved '$ip_or_dns' to $target_ip"
fi

read -r -d '' hostname_patch << EOF
{
  "spec": {
    "template": {
      "spec": {
        "hostAliases": [
          {"ip": "$target_ip", "hostnames": ["$alias"]}
        ]
      }
    }
  }
}
EOF

kubectl -n stackrox patch deployment "$deployment_name" -p "$hostname_patch"
