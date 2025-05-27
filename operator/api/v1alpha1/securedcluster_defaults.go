/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"dario.cat/mergo"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func securedClusterDefaultsToUnstructured(securedCluster *SecuredCluster) (map[string]interface{}, error) {
	defaults, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&securedCluster.Defaults)
	if err != nil {
		return nil, err
	}
	return defaults, nil
}

// AddSecuredClusterDefaultsToUnstructured adds the defaults from SecuredCluster.Defaults to the unstructured object.
func AddSecuredClusterDefaultsToUnstructured(u *unstructured.Unstructured, securedCluster *SecuredCluster) error {
	defaults, err := securedClusterDefaultsToUnstructured(securedCluster)
	if err != nil {
		return err
	}
	u.Object["defaults"] = defaults
	return nil
}

// AddUnstructuredDefaultsToSecuredCluster adds the defaults from the unstructured object to SecuredCluster.
func AddUnstructuredDefaultsToSecuredCluster(securedCluster *SecuredCluster, u *unstructured.Unstructured) error {
	defaultsInterface, found := u.Object["defaults"]
	if !found {
		return nil
	}
	unstructuredDefaults, ok := defaultsInterface.(map[string]interface{})
	if !ok {
		return errors.New("unstructured SecuredCluster defaults of unexpected type")
	}
	typedDefaults := SecuredClusterSpec{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredDefaults, &typedDefaults); err != nil {
		return errors.Wrap(err, "converting defaults from unstructured object into SecuredClusterSpec")
	}
	securedCluster.Defaults = typedDefaults

	return nil
}

// MergeSecuredClusterDefaultsIntoSpec merges the defaults from SecuredCluster.Defaults into SecuredCluster.Spec.
// Modifies content of securedCluster.
func MergeSecuredClusterDefaultsIntoSpec(securedCluster *SecuredCluster) error {
	if err := mergo.Merge(&securedCluster.Spec, securedCluster.Defaults); err != nil {
		return errors.Wrap(err, "merging SecuredCluster Defaults into Spec")
	}
	return nil
}
