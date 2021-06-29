package translation

import (
	"context"
	"testing"

	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
				EnvVars:     map[string]string{"ENV_VAR1": "value6"},
			},
			wantValues: chartutil.Values{
				"labels":      map[string]string{"label1": "value2"},
				"annotations": map[string]string{"annotation1": "value3"},
				"envVars":     map[string]string{"ENV_VAR1": "value6"},
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

func TestGetServiceTLS(t *testing.T) {
	type args struct {
		clientSet  kubernetes.Interface
		serviceTLS *corev1.LocalObjectReference
	}
	tests := map[string]struct {
		args       args
		wantErr    string
		wantValues chartutil.Values
	}{
		"nil": {
			args: args{
				clientSet:  nil,
				serviceTLS: nil,
			},
			wantValues: chartutil.Values{},
		},
		"success": {
			args: args{
				clientSet: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-name", Namespace: "nsname"},
					Data: map[string][]byte{
						"key":  []byte("mock-key"),
						"cert": []byte("mock-cert"),
					},
				}),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantValues: chartutil.Values{
				"serviceTLS": map[string]interface{}{
					"cert": "mock-cert",
					"key":  "mock-key",
				},
			},
		},
		"get-fail": {
			args: args{
				clientSet:  fake.NewSimpleClientset(),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantErr: "failed to retrieve.* secret.* secrets \"secret-name\" not found",
		},
		"key-fail": {
			args: args{
				clientSet: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-name", Namespace: "nsname"},
					Data: map[string][]byte{
						"not-cert": []byte("something else"),
					},
				}),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantErr: "secret \"secret-name\" in namespace \"nsname\".* does not contain member \"key\"",
		},
		"data-fail": {
			args: args{
				clientSet: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-name", Namespace: "nsname"},
					Data: map[string][]byte{
						"key":      []byte("mock-key"),
						"not-cert": []byte("something else"),
					},
				}),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantErr: "secret \"secret-name\" in namespace \"nsname\".* does not contain member \"cert\"",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantNormalized, err := ToHelmValues(tt.wantValues)
			require.NoError(t, err, "error in test specification: cannot normalize want values")
			values, err := GetServiceTLS(context.Background(), tt.args.clientSet, "nsname", tt.args.serviceTLS, "spec.fake.path").Build()
			if tt.wantErr != "" {
				assert.Regexp(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, wantNormalized, values)
			}
		})
	}
}
