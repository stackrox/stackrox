package k8sutil

import (
	"bufio"
	"bytes"
	"io"
	"strings"

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
		return nil, utils.ShouldErr(errors.Errorf("obj was not Unstructured (got %T)", obj))
	}
	return asUnstructured, nil
}

// UnstructuredFromYAMLMulti reads a multi-document YAML string and parses each document
// into an unstructured object. If the document describes a list, the list is flattened,
// i.e., all its objects are added to the result directly.
func UnstructuredFromYAMLMulti(yamlDocStr string) ([]unstructured.Unstructured, error) {
	yamlReader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(yamlDocStr)))

	var result []unstructured.Unstructured
	var yamlDoc []byte
	var err error
	docCounter := 0

	for yamlDoc, err = yamlReader.Read(); err == nil; yamlDoc, err = yamlReader.Read() {
		docCounter++
		if len(bytes.TrimSpace(yamlDoc)) == 0 {
			continue
		}

		jsonDoc, err := yaml.ToJSON(yamlDoc)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert YAML document no. %d to JSON", docCounter)
		}

		obj, _, err := unstructured.UnstructuredJSONScheme.Decode(jsonDoc, nil, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "decoding YAML document no. %d as unstructured", docCounter)
		}
		switch o := obj.(type) {
		case *unstructured.Unstructured:
			result = append(result, *o)
		case *unstructured.UnstructuredList:
			result = append(result, o.Items...)
		default:
			return nil, errors.Errorf("unexpected type %T after decoding YAML document no. %d into unstructured", o, docCounter)
		}
	}
	if err != io.EOF {
		return nil, err
	}
	return result, nil
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
