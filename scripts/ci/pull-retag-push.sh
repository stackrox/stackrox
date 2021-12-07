#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

die() {
  echo >&2 "$@"
  exit 1
}

usage() {
  echo "Usage: $0 --image <registry/image:tag> [FLAGS]"
  echo "FLAGS"
  echo "  --change-registry <reg>: Sets target Docker registry to <reg> "
  echo "  --add-suffix <suffix>:   Appends <suffix> to the retagged image name"
  echo "  --retag <new_tag>:       Changes tag of the image to <new_tag>"
}

SOURCE_IMAGE=""
NEW_REGISTRY=""
IMG_SUFFIX=""
NEW_TAG=""

while [[ "$#" -gt 0 ]]; do case $1 in
        --image) SOURCE_IMAGE="$2"; shift;;
        --change-registry) NEW_REGISTRY="$2"; shift;;
        --add-suffix) IMG_SUFFIX="$2"; shift;;
        --retag) NEW_TAG="$2"; shift;;
        *) { usage; die "Unknown parameter passed: $1"; shift;}
    esac
    shift
done

[[ -n $SOURCE_IMAGE ]] || { usage; die "Missing parameter: --image"; }
[[ "$SOURCE_IMAGE" == */*:* ]] || { usage; die "Invalid format of --image parameter. Expected: registry/image:tag"; }

if [[ -z $NEW_REGISTRY ]] && [[ -z $IMG_SUFFIX ]] && [[ -z $NEW_TAG ]]; then
  echo "There is nothing to change"
  exit 0
fi

[[ $SOURCE_IMAGE =~ ^(.*)/(.*):(.*)$ ]]
REG="${NEW_REGISTRY:-${BASH_REMATCH[1]}}"
IMG="${BASH_REMATCH[2]}${IMG_SUFFIX}"
TAG="${NEW_TAG:-${BASH_REMATCH[3]}}"

echo "Retagging: '${SOURCE_IMAGE}' -> '${REG}/${IMG}:${TAG}'"

docker pull "${SOURCE_IMAGE}"
docker tag "${SOURCE_IMAGE}" "${REG}/${IMG}:${TAG}"
"${DIR}/../ci/push-as-manifest-list.sh" "${REG}/${IMG}:${TAG}"
