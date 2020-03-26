#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export_image() {
  local name=$1
  local image=$2
  local last_dir=$3

  docker pull "$image" | cat

  mkdir -p "${DIR}/${last_dir}"
  echo "Saving $image to ${DIR}/${last_dir}/${name}.img"
  docker save "$image" -o "${DIR}/${last_dir}/${name}.img"
}

save() {
  local registry=$1
  local name=$2
  local tag=$3
  local last_dir=$4

  export_image "$name" "${registry}/${name}:${tag}" "${last_dir}"
}

save_with_rhel() {
  local registry=$1
  local name=$2
  local tag=$3
  local last_dir=$4

  save "$registry" "$name" "$tag" "${last_dir}"
  export_image "$name" "${registry}/${name}-rhel:${tag}" "${last_dir}-rhel"
}

bundle() {
  local name=$1
  pushd "${DIR}"
  tar -czvf "${name}.tgz" "${name}"
  tar -czvf "${name}-rhel.tgz" "${name}-rhel"
  popd
}

store_roxctl() {
  output_path=$1
  gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/darwin"  "${DIR}/${output_path}/bin/darwin"
  gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/linux"   "${DIR}/${output_path}/bin/linux"
  gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/windows" "${DIR}/${output_path}/bin/windows"
  chmod +x "${DIR}/${output_path}/bin/darwin/roxctl" "${DIR}/${output_path}/bin/linux/roxctl"
}

main() {
    # Main uses the version reported by make tag.
    local main_tag="$(make --quiet tag)"
    save_with_rhel "stackrox.io" "main" "${main_tag}" "image-bundle"

    # Scanner uses the version contained in the SCANNER_VERSION file.
    local scanner_tag="$(cat SCANNER_VERSION)"
    save_with_rhel "stackrox.io" "scanner" "${scanner_tag}" "image-bundle"
    save_with_rhel "stackrox.io" "scanner-db" "${scanner_tag}" "image-bundle"

    # The docs image (only advertised offline) uses the release tag (same as Main).
    local docs_tag=${main_tag}
    save_with_rhel "stackrox.io" "docs" "${docs_tag}" "image-bundle"

    # Collector uses the version contained in the COLLECTOR_VERSION file.
    local collector_tag="$(cat COLLECTOR_VERSION)"
    save_with_rhel "collector.stackrox.io" "collector" "${collector_tag}-latest" "image-collector-bundle"

    store_roxctl "image-bundle"
    store_roxctl "image-bundle-rhel"

    bundle "image-bundle"
    bundle "image-collector-bundle"
}

main "$@"
