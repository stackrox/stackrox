#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
set -x
bash ${CODEGEN_PKG}/generate-internal-groups.sh deepcopy \
  github.com/stackrox/stackrox/apis/generated github.com/stackrox/stackrox/apis github.com/stackrox/stackrox/apis \
  "authprovider:v1beta1" \
  --output-base "$(dirname ${BASH_SOURCE})/../../../../.." \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt

# To use your own boilerplate text use:
#   --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt

$GOPATH/src/k8s.io/code-generator/generate-internal-groups.sh deepcopy github.com/stackrox/stackrox/apis/generated github.com/stackrox/stackrox/apis github.com/stackrox/stackrox/apis authprovider:v1beta1 --output-base ./hack/../../../../.. --go-header-file ./hack/../hack/boilerplate.go.txt

 /Users/simon/go/bin/deepcopy-gen --input-dirs github.com/stackrox/stackrox/apis/authprovider,github.com/stackrox/stackrox/apis/authprovider/v1beta1 -O zz_generated.deepcopy --output-base ./hack/../../../../.. --go-header-file ./hack/../hack/boilerplate.go.txt
