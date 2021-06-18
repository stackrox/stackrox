#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

die() {
  echo >&2 "$@"
  exit 1
}

[[ "$#" -eq 2 ]] || die "Usage: $0 <name> <registry>"

name="$1"
[[ -n "$name" ]] || die "No name specified"
registry="$2"
[[ -n "$registry" ]] || die "No registry specified"

registry_auth="$("${DIR}/docker-auth.sh" -m k8s "$registry")"
[[ -n "$registry_auth" ]] || die "Unable to get registry auth info."

cat <<EOF
apiVersion: v1
data:
  .dockerconfigjson: ${registry_auth}
kind: Secret
metadata:
  name: $name
  labels:
    app.kubernetes.io/name: stackrox
type: kubernetes.io/dockerconfigjson
EOF
