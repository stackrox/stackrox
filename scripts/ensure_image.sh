#!/usr/bin/env bash

## This script takes in an image, as well as instructions on how to build it (via a Dockerfile and a build dir).
## First, it tries to pull the image. If it succeeds, it exits.
## If not, it builds the image. If run in CI, it also pushes the image.

die() {
  echo >&2 "$@"
  exit 1
}

image="$1"
dockerfile="$2"
dir="$3"

[[ -n "${image}" && -n "${dockerfile}" && -n "${dir}" ]] || die "Usage $0 <image> <dockerfile_path> <dir>"

echo "Potentially pulling image ${image}"
docker_pull_output="$(docker pull "${image}" 2>&1)"
if [[ "$?" -eq 0 ]]; then
  echo "Image exists. Exiting..."
  exit 0
fi
if [[ ! "${docker_pull_output}" =~ ^.*manifest\ for.*not\ found.*$ ]]; then
  die "Unexpected docker pull error: ${docker_pull_output}"
fi

set -e
echo "Building the image since it doesn't exist"
docker build -t "${image}" -f "${dockerfile}" "${dir}"
if [[ -n "${CI}" ]]; then
    case "$image" in
        quay.io/rhacs-eng/*)
            docker login -u "$QUAY_RHACS_ENG_RW_USERNAME" --password-stdin <<<"$QUAY_RHACS_ENG_RW_PASSWORD" quay.io
            ;;
        quay.io/stackrox-io/*)
            docker login -u "$QUAY_STACKROX_IO_RW_USERNAME" --password-stdin <<<"$QUAY_STACKROX_IO_RW_PASSWORD" quay.io
            ;;
        *)
            die "Unsupported registry of image: $image"
    esac
    docker push "${image}" | cat
else
  echo "Not in CI, not pushing the new image..."
fi
