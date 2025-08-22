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

func centralDefaultsToUnstructured(central *Central) (map[string]interface{}, error) {
	defaults, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&central.Defaults)
	if err != nil {
		return nil, err
	}
	return defaults, nil
}

// AddCentralDefaultsToUnstructured adds the defaults from Central.Defaults to the unstructured object.
func AddCentralDefaultsToUnstructured(u *unstructured.Unstructured, central *Central) error {
	defaults, err := centralDefaultsToUnstructured(central)
	if err != nil {
		return err
	}
	u.Object["defaults"] = defaults
	return nil
}

// AddUnstructuredDefaultsToCentral adds the defaults from the unstructured object to Central.
func AddUnstructuredDefaultsToCentral(central *Central, u *unstructured.Unstructured) error {
	defaultsInterface, found := u.Object["defaults"]
	if !found {
		return nil
	}
	unstructuredDefaults, ok := defaultsInterface.(map[string]interface{})
	if !ok {
		return errors.New("unstructured Central defaults of unexpected type")
	}
	typedDefaults := CentralSpec{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredDefaults, &typedDefaults); err != nil {
		return errors.Wrap(err, "converting defaults from unstructured object into CentralSpec")
	}
	central.Defaults = typedDefaults

	return nil
}

// MergeCentralDefaultsIntoSpec merges the defaults from Central.Defaults into Central.Spec.
// Modifies content of central.
func MergeCentralDefaultsIntoSpec(central *Central) error {
	// Delete old "Default" values, which were previously injected into CRs via CRD defaults.
	// Necessary for the below merging to be effectful in the sense that spec paths in the custom resource
	// with explicit "Default" values are actually filled in with our runtime defaults.
	if scannerV4 := central.Spec.ScannerV4; scannerV4 != nil {
		if scannerComponent := scannerV4.ScannerComponent; scannerComponent != nil && *scannerComponent == "Default" {
			scannerV4.ScannerComponent = nil
		}
	}
	if err := mergo.Merge(&central.Spec, central.Defaults); err != nil {
		return errors.Wrap(err, "merging Central Defaults into Spec")
	}
	return nil
}
