package utils

import (
	"testing"

	pkgLabels "github.com/stackrox/rox/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestShouldAdoptResource(t *testing.T) {
	tests := []struct {
		name     string
		obj      metav1.Object
		expected bool
	}{
		{
			name: "should adopt - has operator managed-by label, no ownerReferences",
			obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-ns",
					Labels: map[string]string{
						pkgLabels.ManagedByLabelKey: pkgLabels.ManagedByOperator,
					},
				},
			},
			expected: true,
		},
		{
			name: "should not adopt - no labels",
			obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-ns",
				},
			},
			expected: false,
		},
		{
			name: "should not adopt - managed by sensor",
			obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-ns",
					Labels: map[string]string{
						pkgLabels.ManagedByLabelKey: pkgLabels.ManagedBySensor,
					},
				},
			},
			expected: false,
		},
		{
			name: "should not adopt - has ownerReference",
			obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-ns",
					Labels: map[string]string{
						pkgLabels.ManagedByLabelKey: pkgLabels.ManagedByOperator,
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "SomeOwner",
							Name:       "owner",
							UID:        types.UID("some-uid"),
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldAdoptResource(tt.obj)
			assert.Equal(t, tt.expected, result)
		})
	}
}
