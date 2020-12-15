#!/usr/bin/env bash
# Flag service protos files which include "google/api/annotations.proto" but do not have their file names end with _service.proto
all_protos=$(git ls-files *.proto)
IFS=$'\n' read -d '' -r -a all_service_protos_without_service_in_name < <(
  git grep -l 'import .*"google/api/annotations.proto";' -- '*.proto' | grep -vE '_service\.proto$'
)

[[ "${#all_service_protos_without_service_in_name[@]}" == 0 ]] || {
  echo "Found service proto files that do not end with _service.proto"
  echo "Files were: "
  printf "  %s\n" ${all_service_protos_without_service_in_name[@]}
  exit 1
} >&2 
