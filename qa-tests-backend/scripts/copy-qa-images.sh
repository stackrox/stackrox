#!/usr/bin/env bash

if [[ "$1" == "" ]]; then
  echo "Usage: $0 path/to/images-to-prefetch.txt"
  exit 1
fi

if [[ ! -f "$1" ]]; then
  echo "File not found: $1"
  exit 1
fi

IMAGE_REPORT=${IMAGE_REPORT:-false}

function copy_qa_image() {
  local full_image="$1"
  if [[ -z "$full_image" ]]; then
    echo "Skipping empty image"
    return
  fi
  local registry
  local repo
  local tag
  local sha
  registry="$(cut -d/ -f1 <<< "$full_image")"
  repo="$(cut -d/ -f3 <<< "$full_image" | cut -d: -f1)"
  tag="$(cut -d: -f2 <<< "$full_image" | cut -d@ -f1)"
  sha="$(cut -d@ -f2 <<< "$full_image")"
  if [[ "$sha" != "$full_image" && -z "$tag" ]]; then
      # A digest-only source needs a destination tag. The digest remains usable
      # from the destination repository after the manifest is copied.
      tag="sha-${sha:7:12}"
  fi
  local new_image="$registry/$QUAY_ORGANIZATION/$repo:$tag"

  if [[ $IMAGE_REPORT == "true" ]]; then
      echo "$repo"
      return
  fi

  skopeo copy --all "docker://$full_image" "docker://$new_image"
}

while read -r line; do
  trimmed=$(echo "$line" | tr -d '[:space:]')
  if [[ -z "$trimmed" ]]; then
    continue
  fi
  if [[ "${trimmed:0:1}" == "#" ]]; then
    continue
  fi

  copy_qa_image "$trimmed"
done < "$1"
