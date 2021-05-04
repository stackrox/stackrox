// Copyright 2018 The Operator-SDK Authors
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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// SupportsOwnerReference checks whether a given dependent supports owner references, based on the owner.
// This function performs following checks:
//  -- True: Owner is cluster-scoped.
//  -- True: Both Owner and dependent are Namespaced with in same namespace.
//  -- False: Owner is Namespaced and dependent is Cluster-scoped.
//  -- False: Both Owner and dependent are Namespaced with different namespaces.
func SupportsOwnerReference(restMapper meta.RESTMapper, owner, dependent runtime.Object) (bool, error) {
	ownerGVK := owner.GetObjectKind().GroupVersionKind()
	ownerMapping, err := restMapper.RESTMapping(ownerGVK.GroupKind(), ownerGVK.Version)
	if err != nil {
		return false, err
	}
	mOwner, err := meta.Accessor(owner)
	if err != nil {
		return false, err
	}

	depGVK := dependent.GetObjectKind().GroupVersionKind()
	depMapping, err := restMapper.RESTMapping(depGVK.GroupKind(), depGVK.Version)
	if err != nil {
		return false, err
	}
	mDep, err := meta.Accessor(dependent)
	if err != nil {
		return false, err
	}
	ownerClusterScoped := ownerMapping.Scope.Name() == meta.RESTScopeNameRoot
	ownerNamespace := mOwner.GetNamespace()
	depClusterScoped := depMapping.Scope.Name() == meta.RESTScopeNameRoot
	depNamespace := mDep.GetNamespace()

	if ownerClusterScoped {
		return true, nil
	}

	if depClusterScoped {
		return false, nil
	}

	if ownerNamespace != depNamespace {
		return false, nil
	}
	// Both owner and dependent are namespace-scoped and in the same namespace.
	return true, nil
}
