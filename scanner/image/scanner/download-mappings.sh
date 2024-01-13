#!/bin/bash

set -euo pipefail

output_dir="/mappings"
mkdir $output_dir

curl --retry 3 -sS --fail -o "${output_dir}/repository-to-cpe.json" https://access.redhat.com/security/data/metrics/repository-to-cpe.json
curl --retry 3 -sS --fail -o "${output_dir}/container-name-repos-map.json" https://access.redhat.com/security/data/metrics/container-name-repos-map.json
