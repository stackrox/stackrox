package extensions

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestValidateDeploymentDefaultsExtension(t *testing.T) {
	tests := map[string]struct {
		obj         *unstructured.Unstructured
		expectError bool
	}{
		"valid configuration": {
			obj: createUnstructuredWithCustomize(&platform.CustomizeSpec{
				DeploymentDefaults: &platform.DeploymentDefaultsSpec{
					NodeSelector: map[string]string{
						"global-node-selector-label1": "global-node-selector-value1",
					},
					Tolerations: []*corev1.Toleration{
						{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
					},
				},
			}),
			expectError: false,
		},
		"invalid: pinToNodes with nodeSelector": {
			obj: createUnstructuredWithCustomize(&platform.CustomizeSpec{
				DeploymentDefaults: &platform.DeploymentDefaultsSpec{
					PinToNodes: ptr.To(platform.PinToNodesInfraRole),
					NodeSelector: map[string]string{
						"global-node-selector-label1": "global-node-selector-value1",
					},
				},
			}),
			expectError: true,
		},
		"invalid: pinToNodes with tolerations": {
			obj: createUnstructuredWithCustomize(&platform.CustomizeSpec{
				DeploymentDefaults: &platform.DeploymentDefaultsSpec{
					PinToNodes: ptr.To(platform.PinToNodesInfraRole),
					Tolerations: []*corev1.Toleration{
						{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
					},
				},
			}),
			expectError: true,
		},
		"invalid: pinToNodes with both nodeSelector and tolerations": {
			obj: createUnstructuredWithCustomize(&platform.CustomizeSpec{
				DeploymentDefaults: &platform.DeploymentDefaultsSpec{
					PinToNodes: ptr.To(platform.PinToNodesInfraRole),
					NodeSelector: map[string]string{
						"global-node-selector-label1": "global-node-selector-value1",
					},
					Tolerations: []*corev1.Toleration{
						{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
					},
				},
			}),
			expectError: true,
		},
		"skips validation for deleted objects": {
			obj: func() *unstructured.Unstructured {
				u := createUnstructuredWithCustomize(&platform.CustomizeSpec{
					DeploymentDefaults: &platform.DeploymentDefaultsSpec{
						PinToNodes: ptr.To(platform.PinToNodesInfraRole),
						NodeSelector: map[string]string{
							"global-node-selector-label1": "global-node-selector-value1",
						},
					},
				})
				now := metav1.NewTime(time.Now())
				u.SetDeletionTimestamp(&now)
				return u
			}(),
			expectError: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ext := ValidateDeploymentDefaultsExtension()
			err := ext(context.Background(), tt.obj, nil, logr.Discard())

			if tt.expectError {
				assert.Error(t, err, "expected validation to fail")
			} else {
				assert.NoError(t, err, "expected validation to pass")
			}
		})
	}
}

func createUnstructuredWithCustomize(customize *platform.CustomizeSpec) *unstructured.Unstructured {
	central := &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "stackrox",
		},
		Spec: platform.CentralSpec{
			Customize: customize,
		},
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(central)
	if err != nil {
		panic(err)
	}
	return &unstructured.Unstructured{Object: obj}
}
