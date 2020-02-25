package k8sutil

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// UnstructuredFromYAML takes a Kube YAML object and returns an Unstructured struct.
func UnstructuredFromYAML(yamlStr string) (*unstructured.Unstructured, error) {
	jsonBytes, err := yaml.ToJSON([]byte(yamlStr))
	if err != nil {
		return nil, errors.Wrap(err, "converting to JSON")
	}

	obj, _, err := unstructured.UnstructuredJSONScheme.Decode(jsonBytes, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "decoding as unstructured")
	}
	asUnstructured, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, utils.Should(errors.Errorf("obj was not Unstructured (got %T)", obj))
	}
	return asUnstructured, nil
}

// DeleteAnnotation deletes the given annotation from the object.
// If this is the last remaining annotation, it sets the annotations to nil
// which ends up clearing the field.
func DeleteAnnotation(object *unstructured.Unstructured, key string) {
	existingAnns := object.GetAnnotations()
	delete(existingAnns, key)
	if len(existingAnns) == 0 {
		existingAnns = nil
	}
	object.SetAnnotations(existingAnns)
}

// SetAnnotation sets an annotation on the given object.
// Any existing value for the same key is clobbered.
func SetAnnotation(object *unstructured.Unstructured, key, value string) {
	anns := object.GetAnnotations()
	if anns == nil {
		anns = make(map[string]string)
	}
	anns[key] = value
	object.SetAnnotations(anns)
}
