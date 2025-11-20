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
			obj: toUnstructured(t, centralWithDeploymentDefaults(&platform.DeploymentDefaultsSpec{
				NodeSelector: map[string]string{
					"global-node-selector-label": "global-node-selector-value",
				},
				Tolerations: []*corev1.Toleration{
					{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
				},
			})),
			expectError: false,
		},
		"invalid: pinToNodes with nodeSelector": {
			obj: toUnstructured(t, centralWithDeploymentDefaults(&platform.DeploymentDefaultsSpec{
				PinToNodes: ptr.To(platform.PinToNodesInfraRole),
				NodeSelector: map[string]string{
					"global-node-selector-label": "global-node-selector-value",
				},
			})),
			expectError: true,
		},
		"invalid: pinToNodes with tolerations": {
			obj: toUnstructured(t, centralWithDeploymentDefaults(&platform.DeploymentDefaultsSpec{
				PinToNodes: ptr.To(platform.PinToNodesInfraRole),
				Tolerations: []*corev1.Toleration{
					{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
				},
			})),
			expectError: true,
		},
		"invalid: pinToNodes with both nodeSelector and tolerations": {
			obj: toUnstructured(t, centralWithDeploymentDefaults(&platform.DeploymentDefaultsSpec{
				PinToNodes: ptr.To(platform.PinToNodesInfraRole),
				NodeSelector: map[string]string{
					"global-node-selector-label": "global-node-selector-value",
				},
				Tolerations: []*corev1.Toleration{
					{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
				},
			})),
			expectError: true,
		},
		"skips validation for deleted objects": {
			obj: func() *unstructured.Unstructured {
				central := centralWithDeploymentDefaults(&platform.DeploymentDefaultsSpec{
					PinToNodes: ptr.To(platform.PinToNodesInfraRole),
					NodeSelector: map[string]string{
						"global-node-selector-label": "global-node-selector-value",
					},
				})
				now := metav1.NewTime(time.Now())
				central.SetDeletionTimestamp(&now)
				return toUnstructured(t, central)
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

func centralWithDeploymentDefaults(d *platform.DeploymentDefaultsSpec) *platform.Central {
	return &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "stackrox",
		},
		Spec: platform.CentralSpec{
			Customize: &platform.CustomizeSpec{
				DeploymentDefaults: d,
			},
		},
	}
}

func toUnstructured(t *testing.T, obj runtime.Object) *unstructured.Unstructured {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	assert.NoError(t, err)
	return &unstructured.Unstructured{Object: u}
}
