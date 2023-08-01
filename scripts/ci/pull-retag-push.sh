#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"

set -euxo pipefail

if (( "$#" != 2 )); then
  die "Usage: $0 SRC DEST"
fi

SRC="$1"
DEST="$2"

[[ "${OPENSHIFT_CI:-false}" == "false" ]] || { die "Not supported in OpenShift CI"; }

docker pull "${SRC}" | cat
docker tag "${SRC}" "${DEST}"
docker push "${DEST}" | cat
