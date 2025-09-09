package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFromUnstructuredToSpecificTypePointer(t *testing.T) {
	t.Run("should error if 'from' is not an unstructured object", func(tt *testing.T) {
		from := &corev1.Pod{}
		to := &corev1.Pod{}
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(from, to))
	})

	t.Run("should error if 'to' is not a pointer", func(tt *testing.T) {
		from := &unstructured.Unstructured{}
		to := corev1.Pod{}
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(from, to))
	})

	t.Run("should error if 'to' is a nil pointer", func(tt *testing.T) {
		from := &unstructured.Unstructured{}
		var to *corev1.Pod = nil
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(from, to))
	})

	t.Run("successful convertion", func(tt *testing.T) {
		from := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "pod-name",
					"namespace": "pod-namespace",
				},
			},
		}
		to := &corev1.Pod{}
		assert.NoError(tt, FromUnstructuredToSpecificTypePointer(from, to))
		assert.Equal(tt, "pod-name", to.GetName())
		assert.Equal(tt, "pod-namespace", to.GetNamespace())
	})
	t.Run("convertion with unexpected fields", func(tt *testing.T) {
		from := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"unexpected-field": "some-data",
					"name":             "pod-name",
					"namespace":        "pod-namespace",
				},
			},
		}
		to := &corev1.Pod{}
		// Unexpected fields should not yield an error, they are ignored.
		assert.NoError(tt, FromUnstructuredToSpecificTypePointer(from, to))
		assert.Equal(tt, "pod-name", to.GetName())
		assert.Equal(tt, "pod-namespace", to.GetNamespace())
	})
	t.Run("convertion should fail with unexpected value", func(tt *testing.T) {
		from := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      0, // Name is expected to be a string
					"namespace": "pod-namespace",
				},
			},
		}
		to := &corev1.Pod{}
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(from, to))
	})
}
