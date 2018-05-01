#!/bin/bash

function usage() {
  echo "To go get a package at a particular version, use:"
  echo "$0 <package> <hash_or_tag>"
  exit 1
}

[[ "$#" -eq 2 ]] || usage

package="$1"
hash_or_tag="$2"

package_without_trailing_dots="${package%/...}"
go get -d "${package_without_trailing_dots}"
cd "${GOPATH}/src/${package_without_trailing_dots}"

git rev-parse --git-dir > /dev/null 2>&1 || { echo "This script only supports git-based packages!"; exit 1; }
git checkout -q "${hash_or_tag}"
go install "${package}"