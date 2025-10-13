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

bundle() {
  local name=$1
  pushd "${DIR}"
  tar -czvf "${name}.tgz" "${name}"
  popd
}

store_roxctl() {
  output_path=$1
  mkdir -p "${DIR}/${output_path}/bin"
  gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/darwin"  "${DIR}/${output_path}/bin"
  gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/linux"   "${DIR}/${output_path}/bin"
  gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/windows" "${DIR}/${output_path}/bin"
  chmod +x "${DIR}/${output_path}/bin/darwin/roxctl" "${DIR}/${output_path}/bin/linux/roxctl"
}

main() {
    # Main uses the version reported by make tag.
    local main_tag
    main_tag="$(make --quiet --no-print-directory tag)"
    save "stackrox.io" "main" "${main_tag}" "image-bundle"

    # Scanner uses the same version as Main.
    save "stackrox.io" "scanner" "${main_tag}" "image-bundle"
    save "stackrox.io" "scanner-db" "${main_tag}" "image-bundle"

    # Collector uses the same version as Main.
    save "collector.stackrox.io" "collector" "${main_tag}" "image-collector-bundle"

    store_roxctl "image-bundle"

    bundle "image-bundle"
    bundle "image-collector-bundle"
}

main "$@"
