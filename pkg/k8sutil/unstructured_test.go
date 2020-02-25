package k8sutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	testYAMLNoAnnotations = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackrox
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`

	testYAMLOneAnnotation = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackrox
  annotations:
    foo: bar
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`

	testYAMLTwoAnnotations = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackrox
  annotations:
    foo: bar
    foo2: bar2
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`
)

func fromYAML(yaml string, t *testing.T) *unstructured.Unstructured {
	obj, err := UnstructuredFromYAML(yaml)
	require.NoError(t, err)
	return obj
}

func assertObjEqualsYAML(obj *unstructured.Unstructured, yaml string, t *testing.T) {
	assert.Equal(t, fromYAML(yaml, t), obj)
}

func TestAnnotations(t *testing.T) {
	noAnnotationsMutable := fromYAML(testYAMLNoAnnotations, t)
	oneAnnotationMutable := fromYAML(testYAMLOneAnnotation, t)
	twoAnnotationsMutable := fromYAML(testYAMLTwoAnnotations, t)

	assert.NotEqual(t, oneAnnotationMutable, twoAnnotationsMutable)

	DeleteAnnotation(twoAnnotationsMutable, "foo2")
	assertObjEqualsYAML(twoAnnotationsMutable, testYAMLOneAnnotation, t)

	DeleteAnnotation(oneAnnotationMutable, "foo")
	assertObjEqualsYAML(oneAnnotationMutable, testYAMLNoAnnotations, t)

	SetAnnotation(noAnnotationsMutable, "foo", "bar")
	assertObjEqualsYAML(noAnnotationsMutable, testYAMLOneAnnotation, t)

	SetAnnotation(noAnnotationsMutable, "foo2", "bar2")
	assertObjEqualsYAML(noAnnotationsMutable, testYAMLTwoAnnotations, t)
}
