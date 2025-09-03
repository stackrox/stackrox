package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFromUnstructuredToSpecificTypePointer(t *testing.T) {
	t.Run("should error if 'to' is not an unstructured object", func(tt *testing.T) {
		to := &corev1.Pod{}
		from := &corev1.Pod{}
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(to, from))
	})

	t.Run("should error if 'from' is not a pointer", func(tt *testing.T) {
		to := &unstructured.Unstructured{}
		from := corev1.Pod{}
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(to, from))
	})

	t.Run("should error if 'from' is a nil pointer", func(tt *testing.T) {
		to := &unstructured.Unstructured{}
		var from *corev1.Pod = nil
		assert.Error(tt, FromUnstructuredToSpecificTypePointer(to, from))
	})

	t.Run("successful convertion", func(tt *testing.T) {
		to := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "pod-name",
					"namespace": "pod-namespace",
				},
			},
		}
		from := &corev1.Pod{}
		assert.NoError(tt, FromUnstructuredToSpecificTypePointer(to, from))
		assert.Equal(tt, "pod-name", to.GetName())
		assert.Equal(tt, "pod-namespace", to.GetNamespace())
	})
}
