package translation

import (
	"testing"

	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetCustomize(t *testing.T) {
	tests := map[string]struct {
		customizeSpec *common.CustomizeSpec
		values        chartutil.Values
		wantValues    chartutil.Values
	}{
		"nil": {
			customizeSpec: nil,
			wantValues:    chartutil.Values{},
		},
		"empty": {
			customizeSpec: &common.CustomizeSpec{},
			wantValues:    chartutil.Values{},
		},
		"all-data": {
			customizeSpec: &common.CustomizeSpec{
				Labels:      map[string]string{"label1": "value2"},
				Annotations: map[string]string{"annotation1": "value3"},
				EnvVars: []corev1.EnvVar{
					{
						Name:  "ENV_VAR1",
						Value: "value6",
					},
				},
			},
			wantValues: chartutil.Values{
				"labels":      map[string]string{"label1": "value2"},
				"annotations": map[string]string{"annotation1": "value3"},
				"envVars": map[string]interface{}{
					"ENV_VAR1": map[string]interface{}{
						"value": "value6",
					},
				},
			},
		},
		"partial-data": {
			customizeSpec: &common.CustomizeSpec{
				Labels: map[string]string{"value2": "should-apply"},
			},
			wantValues: chartutil.Values{
				"labels": map[string]string{"value2": "should-apply"},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantNormalized, err := ToHelmValues(tt.wantValues)
			require.NoError(t, err, "error in test specification: cannot normalize want values")
			values, err := GetCustomize(tt.customizeSpec).Build()
			assert.NoError(t, err)
			assert.Equal(t, wantNormalized, values)
		})
	}
}

func TestGetResources(t *testing.T) {
	tests := map[string]struct {
		resources  *corev1.ResourceRequirements
		wantValues chartutil.Values
	}{
		"nil": {
			resources:  nil,
			wantValues: chartutil.Values{},
		},
		"nil-override": {
			resources:  &corev1.ResourceRequirements{},
			wantValues: chartutil.Values{},
		},
		"data-full": {
			resources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("1"),
					corev1.ResourceEphemeralStorage: resource.MustParse("3"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("2"),
				},
			},
			wantValues: chartutil.Values{
				"limits": corev1.ResourceList{
					"cpu":               *resource.NewQuantity(1, resource.DecimalSI),
					"ephemeral-storage": *resource.NewQuantity(3, resource.DecimalSI),
				},
				"requests": corev1.ResourceList{
					"memory": *resource.NewQuantity(2, resource.DecimalSI),
				},
			},
		},
		"data-no-limits": {
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("5"),
				},
			},
			wantValues: chartutil.Values{
				"requests": corev1.ResourceList{
					"memory": *resource.NewQuantity(5, resource.DecimalSI),
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantNormalized, err := ToHelmValues(tt.wantValues)
			require.NoError(t, err, "error in test specification: cannot normalize want values")
			values, err := GetResources(tt.resources).Build()
			assert.NoError(t, err)
			assert.Equal(t, wantNormalized, values)
		})
	}
}
