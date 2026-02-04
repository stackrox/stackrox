package common

import (
	"errors"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	ErrorAnnotationsUpdated = errors.New("reconciliation deferred after persisting annotations; will resume automatically")
)

func AnnotationsEqual(unstructuredA, unstructuredB *unstructured.Unstructured) bool {
	annotationsA := unstructuredA.GetAnnotations()
	annotationsB := unstructuredB.GetAnnotations()
	// We use this so that nil and empty map are treated the same.
	if len(annotationsA) == 0 && len(annotationsB) == 0 {
		return true
	}
	return reflect.DeepEqual(annotationsA, annotationsB)
}
