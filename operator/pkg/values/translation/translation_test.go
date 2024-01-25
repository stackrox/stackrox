package translation

import (
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetCustomize(t *testing.T) {
	tests := map[string]struct {
		customizeSpec *platform.CustomizeSpec
		values        chartutil.Values
		wantValues    chartutil.Values
	}{
		"nil": {
			customizeSpec: nil,
			wantValues:    chartutil.Values{},
		},
		"empty": {
			customizeSpec: &platform.CustomizeSpec{},
			wantValues:    chartutil.Values{},
		},
		"all-data": {
			customizeSpec: &platform.CustomizeSpec{
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
			customizeSpec: &platform.CustomizeSpec{
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

func TestGetTLSConfigValues(t *testing.T) {
	tests := map[string]struct {
		tls  *platform.TLSConfig
		want chartutil.Values
	}{
		"nil": {
			tls:  nil,
			want: chartutil.Values{},
		},
		"empty": {
			tls:  &platform.TLSConfig{AdditionalCAs: []platform.AdditionalCA{}},
			want: chartutil.Values{},
		},
		"single-ca": {
			tls: &platform.TLSConfig{
				AdditionalCAs: []platform.AdditionalCA{
					{
						Name:    "ca-name",
						Content: "ca-content",
					},
				},
			},
			want: chartutil.Values{
				"additionalCAs": map[string]interface{}{
					"ca-name": "ca-content",
				},
			},
		},
		"many-cas": {
			tls: &platform.TLSConfig{
				AdditionalCAs: []platform.AdditionalCA{
					{
						Name:    "ca1-name",
						Content: "ca1-content",
					},
					{
						Name:    "ca2-name",
						Content: "ca2-content",
					},
				},
			},
			want: chartutil.Values{
				"additionalCAs": map[string]interface{}{
					"ca1-name": "ca1-content",
					"ca2-name": "ca2-content",
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			values, err := GetTLSConfigValues(tt.tls).Build()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, values)
		})
	}
}

func TestSetScannerComponentDisableValue(t *testing.T) {
	tests := map[string]struct {
		scannerComponent platform.ScannerComponentPolicy
		want             chartutil.Values
		wantErr          bool
	}{
		"Disabled": {
			scannerComponent: platform.ScannerComponentDisabled,
			want: chartutil.Values{
				"disable": true,
			},
		},
		"Enabled": {
			scannerComponent: platform.ScannerComponentEnabled,
			want: chartutil.Values{
				"disable": false,
			},
		},
		"Invalid": {
			scannerComponent: "invalid",
			want:             chartutil.Values{},
			wantErr:          true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			vb := NewValuesBuilder()
			scannerComponent := tt.scannerComponent
			SetScannerComponentDisableValue(&vb, &scannerComponent)
			values, err := vb.Build()
			if tt.wantErr {
				require.NotNil(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, values)
		})
	}
}

func TestSetScannerV4ComponentValues(t *testing.T) {
	// using local copies to be able to create references to those constants
	autoscalingEnabled := platform.ScannerAutoScalingEnabled
	autoscalingDisabled := platform.ScannerAutoScalingDisabled
	autoscalingInvalid := platform.AutoScalingPolicy("invalid")

	tests := map[string]struct {
		component    *platform.ScannerV4Component
		componentKey string
		want         chartutil.Values
		wantErr      bool
	}{
		"empty for default component": {
			component:    &platform.ScannerV4Component{},
			componentKey: "indexer",
			want:         chartutil.Values{},
		},
		"sets resources": {
			component: &platform.ScannerV4Component{
				DeploymentSpec: platform.DeploymentSpec{
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("200M"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("150m"),
							corev1.ResourceMemory: resource.MustParse("250M"),
						},
					},
				},
			},
			componentKey: "indexer",
			want: chartutil.Values{
				"indexer": map[string]interface{}{
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "100m",
							"memory": "200M",
						},
						"limits": map[string]interface{}{
							"cpu":    "150m",
							"memory": "250M",
						},
					},
				},
			},
		},
		"uses given input componentKey as toplevel key": {
			component: &platform.ScannerV4Component{
				DeploymentSpec: platform.DeploymentSpec{
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("100m"),
						},
					},
				},
			},
			componentKey: "matcher",
			want: chartutil.Values{
				"matcher": map[string]interface{}{
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu": "100m",
						},
					},
				},
			},
		},
		"sets autoscaling if enabled": {
			component: &platform.ScannerV4Component{
				Scaling: &platform.ScannerComponentScaling{
					// using a local copy of platform.ScannerAutoScalingEnabled in order to pass it as a reference
					// since references to const strings are not allowed in Go
					AutoScaling: &autoscalingEnabled,
					MinReplicas: pointers.Int32(1),
					MaxReplicas: pointers.Int32(3),
				},
			},
			componentKey: "indexer",
			want: chartutil.Values{
				"indexer": map[string]interface{}{
					"autoscaling": map[string]interface{}{
						"disable":     false,
						"minReplicas": int32(1),
						"maxReplicas": int32(3),
					},
				},
			},
		},
		"autoscaling can be disabled": {
			component: &platform.ScannerV4Component{
				Scaling: &platform.ScannerComponentScaling{
					AutoScaling: &autoscalingDisabled,
				},
			},
			componentKey: "indexer",
			want: chartutil.Values{
				"indexer": map[string]interface{}{
					"autoscaling": map[string]interface{}{
						"disable": true,
					},
				},
			},
		},
		"err for invalid AutoscalingPolicy": {
			component: &platform.ScannerV4Component{
				Scaling: &platform.ScannerComponentScaling{
					AutoScaling: &autoscalingInvalid,
				},
			},
			wantErr: true,
		},
		"set replicas if available": {
			component: &platform.ScannerV4Component{
				Scaling: &platform.ScannerComponentScaling{
					Replicas: pointers.Int32(2),
				},
			},
			componentKey: "indexer",
			want: chartutil.Values{
				"indexer": map[string]interface{}{
					"replicas": int32(2),
				},
			},
		},
		"set tolerations": {
			component: &platform.ScannerV4Component{
				DeploymentSpec: platform.DeploymentSpec{
					Tolerations: []*corev1.Toleration{
						{Key: "masternode", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
					},
				},
			},
			componentKey: "indexer",
			want: chartutil.Values{
				"indexer": map[string]interface{}{
					"tolerations": []interface{}{
						map[string]interface{}{
							"effect": "NoSchedule", "key": "masternode", "operator": "Exists",
						},
					},
				},
			},
		},
		"set nodeSelector": {
			component: &platform.ScannerV4Component{
				DeploymentSpec: platform.DeploymentSpec{
					NodeSelector: map[string]string{
						"masternode": "true",
					},
				},
			},
			componentKey: "indexer",
			want: chartutil.Values{
				"indexer": map[string]interface{}{
					"nodeSelector": map[string]interface{}{
						"masternode": "true",
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			vb := NewValuesBuilder()
			SetScannerV4ComponentValues(&vb, tt.componentKey, tt.component)
			values, err := vb.Build()
			if tt.wantErr {
				require.NotNil(t, err)
				return
			}
			assert.NoError(t, err)

			// This is done in order to prevent mismatch of number types
			// in values in case the helm dependency does not parse correctly
			// specifically int32's were parsed as float64's
			wantAsValues, err := ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			assert.Equal(t, wantAsValues, values)
		})
	}
}

func TestSetScannerV4DBValues(t *testing.T) {
	tests := map[string]struct {
		db      *platform.ScannerV4DB
		want    chartutil.Values
		wantErr bool
	}{
		"empty for default component": {
			db:   &platform.ScannerV4DB{},
			want: chartutil.Values{},
		},
		"sets resources": {
			db: &platform.ScannerV4DB{
				DeploymentSpec: platform.DeploymentSpec{
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("200M"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("150m"),
							corev1.ResourceMemory: resource.MustParse("250M"),
						},
					},
				},
			},
			want: chartutil.Values{
				"db": map[string]interface{}{
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "100m",
							"memory": "200M",
						},
						"limits": map[string]interface{}{
							"cpu":    "150m",
							"memory": "250M",
						},
					},
				},
			},
		},
		"set tolerations": {
			db: &platform.ScannerV4DB{
				DeploymentSpec: platform.DeploymentSpec{
					Tolerations: []*corev1.Toleration{
						{Key: "masternode", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
					},
				},
			},
			want: chartutil.Values{
				"db": map[string]interface{}{
					"tolerations": []interface{}{
						map[string]interface{}{
							"effect": "NoSchedule", "key": "masternode", "operator": "Exists",
						},
					},
				},
			},
		},
		"set nodeSelector": {
			db: &platform.ScannerV4DB{
				DeploymentSpec: platform.DeploymentSpec{
					NodeSelector: map[string]string{
						"masternode": "true",
					},
				},
			},
			want: chartutil.Values{
				"db": map[string]interface{}{
					"nodeSelector": map[string]interface{}{
						"masternode": "true",
					},
				},
			},
		},
		"set persistence.persistentVolumeClaim": {
			db: &platform.ScannerV4DB{
				Persistence: &platform.ScannerV4Persistence{
					PersistentVolumeClaim: &platform.ScannerV4PersistentVolumeClaim{
						ClaimName:        pointers.String("test"),
						Size:             pointers.String("100GB"),
						StorageClassName: pointers.String("testSC"),
					},
				},
			},
			want: chartutil.Values{
				"db": map[string]interface{}{
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"claimName":    "test",
							"createClaim":  true,
							"size":         "100GB",
							"storageClass": "testSC",
						},
					},
				},
			},
		},
		"set persistence.hostPath": {
			db: &platform.ScannerV4DB{
				Persistence: &platform.ScannerV4Persistence{
					HostPath: &platform.HostPathSpec{
						Path: pointers.String("/test/path"),
					},
				},
			},
			want: chartutil.Values{
				"db": map[string]interface{}{
					"persistence": map[string]interface{}{
						"hostPath": "/test/path",
					},
				},
			},
		},
		"err for invalid persistence": {
			db: &platform.ScannerV4DB{
				Persistence: &platform.ScannerV4Persistence{
					PersistentVolumeClaim: &platform.ScannerV4PersistentVolumeClaim{
						ClaimName: pointers.String("test"),
					},
					HostPath: &platform.HostPathSpec{
						Path: pointers.String("/test/path"),
					},
				},
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			vb := NewValuesBuilder()
			SetScannerV4DBValues(&vb, tt.db)
			values, err := vb.Build()
			if tt.wantErr {
				require.NotNil(t, err)
				return
			}

			assert.NoError(t, err)

			// This is done in order to prevent mismatch of number types
			// in values in case the helm dependency does not parse correctly
			// specifically int32's were parsed as float64's
			wantAsValues, err := ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			assert.Equal(t, wantAsValues, values)
		})
	}
}
