#!/usr/bin/env bash

set -euxo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

if (( "$#" != 2 )); then
  echo >&2 "Usage: $0 SRC DEST"
  exit 1
fi

SRC="$1"
DEST="$2"

docker pull "${SRC}" | cat
docker tag "${SRC}" "${DEST}"
"${DIR}/push-as-manifest-list.sh" "${DEST}" | cat
