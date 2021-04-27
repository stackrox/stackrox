// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sutil

import (
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/version"
)

type CRDVersions []apiextv1beta1.CustomResourceDefinitionVersion

func (vs CRDVersions) Len() int { return len(vs) }
func (vs CRDVersions) Less(i, j int) bool {
	return version.CompareKubeAwareVersionStrings(vs[i].Name, vs[j].Name) > 0
}
func (vs CRDVersions) Swap(i, j int) { vs[i], vs[j] = vs[j], vs[i] }
