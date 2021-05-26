package translation

import (
	"context"
	"testing"

	common "github.com/stackrox/rox/operator/common/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSetBool(t *testing.T) {
	type args struct {
		b      *bool
		label  string
		values chartutil.Values
	}
	truth := true
	falsehood := false
	tests := map[string]struct {
		args       args
		wantValues chartutil.Values
	}{
		"nil": {
			args: args{
				b:      nil,
				label:  "a",
				values: chartutil.Values{},
			},
			wantValues: chartutil.Values{},
		},
		"nil-no-override": {
			args: args{
				b:      nil,
				label:  "a",
				values: chartutil.Values{"a": true},
			},
			wantValues: chartutil.Values{"a": true},
		},
		"false": {
			args: args{
				b:      &falsehood,
				label:  "a",
				values: chartutil.Values{},
			},
			wantValues: chartutil.Values{"a": false},
		},
		"true": {
			args: args{
				b:      &truth,
				label:  "a",
				values: chartutil.Values{},
			},
			wantValues: chartutil.Values{"a": true},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			SetBool(tt.args.b, tt.args.label, tt.args.values)
			assert.Equal(t, tt.wantValues, tt.args.values)
		})
	}
}

func TestSetCustomize(t *testing.T) {
	tests := map[string]struct {
		customizeSpec *common.CustomizeSpec
		values        chartutil.Values
		component     CustomizeComponent
		wantValues    chartutil.Values
	}{
		"nil-top": {
			customizeSpec: nil,
			values:        chartutil.Values{"value1": "should-stay"},
			component:     CustomizeTopLevel,
			wantValues:    chartutil.Values{"value1": "should-stay"},
		},
		"nil-child": {
			customizeSpec: nil,
			values:        chartutil.Values{"value1": "should-stay"},
			component:     CustomizeCollector,
			wantValues:    chartutil.Values{"value1": "should-stay"},
		},
		"empty": {
			customizeSpec: &common.CustomizeSpec{},
			values:        chartutil.Values{"value1": "should-stay"},
			component:     CustomizeTopLevel,
			wantValues:    chartutil.Values{"value1": "should-stay"},
		},
		"data-top": {
			customizeSpec: &common.CustomizeSpec{
				Labels:         map[string]string{"label1": "value2"},
				Annotations:    map[string]string{"annotation1": "value3"},
				PodLabels:      map[string]string{"pod-label1": "value4"},
				PodAnnotations: map[string]string{"pod-annotation1": "value5"},
				EnvVars:        map[string]string{"ENV_VAR1": "value6"},
			},
			values:    chartutil.Values{"value1": "should-stay"},
			component: CustomizeTopLevel,
			wantValues: chartutil.Values{
				"value1": "should-stay",
				"customize": chartutil.Values{
					"labels":         map[string]string{"label1": "value2"},
					"annotations":    map[string]string{"annotation1": "value3"},
					"podLabels":      map[string]string{"pod-label1": "value4"},
					"podAnnotations": map[string]string{"pod-annotation1": "value5"},
					"envVars":        map[string]string{"ENV_VAR1": "value6"},
				},
			},
		},
		"data-top-replace": {
			customizeSpec: &common.CustomizeSpec{
				Labels: map[string]string{"value2": "should-apply"},
			},
			values: chartutil.Values{
				"value1": "should-stay",
				"customize": chartutil.Values{
					"labels":    map[string]string{"this": "should-go"},
					"podLabels": map[string]string{"this": "should-stay-too"},
				},
			},
			component: CustomizeTopLevel,
			wantValues: chartutil.Values{
				"value1": "should-stay",
				"customize": chartutil.Values{
					"labels":    map[string]string{"value2": "should-apply"},
					"podLabels": map[string]string{"this": "should-stay-too"},
				},
			},
		},
		"data-child": {
			customizeSpec: &common.CustomizeSpec{
				Labels:         map[string]string{"label1": "value2"},
				Annotations:    map[string]string{"annotation1": "value3"},
				PodLabels:      map[string]string{"pod-label1": "value4"},
				PodAnnotations: map[string]string{"pod-annotation1": "value5"},
				EnvVars:        map[string]string{"ENV_VAR1": "value6"},
			},
			values:    chartutil.Values{"value1": "should-stay"},
			component: CustomizeCollector,
			wantValues: chartutil.Values{
				"value1": "should-stay",
				"customize": chartutil.Values{
					"collector": chartutil.Values{
						"labels":         map[string]string{"label1": "value2"},
						"annotations":    map[string]string{"annotation1": "value3"},
						"podLabels":      map[string]string{"pod-label1": "value4"},
						"podAnnotations": map[string]string{"pod-annotation1": "value5"},
						"envVars":        map[string]string{"ENV_VAR1": "value6"},
					},
				},
			},
		},
		"data-child-replace": {
			customizeSpec: &common.CustomizeSpec{
				Labels: map[string]string{"value2": "should-apply"},
			},
			values: chartutil.Values{
				"value1": "should-stay",
				"customize": chartutil.Values{
					"collector": chartutil.Values{
						"labels": chartutil.Values{
							"this": "should-be-gone",
						},
						"podLabels": chartutil.Values{
							"this": "should-be-gone-too",
						},
					},
					"sensor": chartutil.Values{
						"labels": chartutil.Values{
							"this": "should-stay",
						},
					},
				},
			},
			component: CustomizeCollector,
			wantValues: chartutil.Values{
				"value1": "should-stay",
				"customize": chartutil.Values{
					"collector": chartutil.Values{
						"labels": map[string]string{"value2": "should-apply"},
					},
					"sensor": chartutil.Values{
						"labels": chartutil.Values{
							"this": "should-stay",
						},
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			SetCustomize(tt.customizeSpec, tt.values, tt.component)
			assert.Equal(t, tt.wantValues, tt.values)
		})
	}
}

func TestSetResources(t *testing.T) {
	tests := map[string]struct {
		resources  *common.Resources
		values     chartutil.Values
		wantValues chartutil.Values
		key        ResourcesKey
	}{
		"nil": {
			resources:  nil,
			values:     chartutil.Values{"asd": "123"},
			wantValues: chartutil.Values{"asd": "123"},
			key:        ResourcesLabel,
		},
		"data-full": {
			resources: &common.Resources{
				Override: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.Quantity{Format: "1"},
						corev1.ResourceEphemeralStorage: resource.Quantity{Format: "3"},
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.Quantity{Format: "2"},
					},
				},
			},
			values: chartutil.Values{
				"asd": "123",
			},
			wantValues: chartutil.Values{
				"complianceResources": chartutil.Values{
					"limits": corev1.ResourceList{
						"cpu":               resource.Quantity{Format: "1"},
						"ephemeral-storage": resource.Quantity{Format: "3"},
					},
					"requests": corev1.ResourceList{
						"memory": resource.Quantity{Format: "2"},
					},
				},
				"asd": "123",
			},
			key: ResourcesComplianceLabel,
		},
		"data-no-limits": {
			resources: &common.Resources{
				Override: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.Quantity{Format: "5"},
					},
				},
			},
			values: chartutil.Values{
				"asd": "123",
			},
			wantValues: chartutil.Values{
				"resources": chartutil.Values{
					"requests": corev1.ResourceList{
						"memory": resource.Quantity{Format: "5"},
					},
				},
				"asd": "123",
			},
			key: ResourcesLabel,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			SetResources(tt.resources, tt.values, tt.key)
			assert.Equal(t, tt.wantValues, tt.values)
		})
	}
}

func TestSetServiceTLS(t *testing.T) {
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
					StringData: map[string]string{
						"key":  "mock-key",
						"cert": "mock-cert",
					},
				}),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantValues: chartutil.Values{
				"serviceTLS": chartutil.Values{
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
			wantErr: "secrets \"secret-name\" not found",
		},
		"key-fail": {
			args: args{
				clientSet: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-name", Namespace: "nsname"},
					StringData: map[string]string{
						"not-cert": "something else",
					},
				}),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantErr: "secret \"secret-name\" in namespace \"nsname\" does not contain member \"key\"",
		},
		"data-fail": {
			args: args{
				clientSet: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-name", Namespace: "nsname"},
					StringData: map[string]string{
						"key":      "mock-key",
						"not-cert": "something else",
					},
				}),
				serviceTLS: &corev1.LocalObjectReference{Name: "secret-name"},
			},
			wantErr: "secret \"secret-name\" in namespace \"nsname\" does not contain member \"cert\"",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			values := make(chartutil.Values)
			err := SetServiceTLS(context.Background(), tt.args.clientSet, "nsname", tt.args.serviceTLS, values)
			if tt.wantErr != "" {
				assert.Regexp(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValues, values)
			}
		})
	}
}
