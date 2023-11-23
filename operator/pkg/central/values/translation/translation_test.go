package translation

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/common"
	"github.com/stackrox/rox/operator/pkg/central/extensions"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	fkClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReadBaseValues(t *testing.T) {
	_, err := chartutil.ReadValues(baseValuesYAML)
	assert.NoError(t, err)
}

func TestTranslate(t *testing.T) {
	type args struct {
		clientSet kubernetes.Interface
		c         platform.Central
		pvcs      []*corev1.PersistentVolumeClaim
		version   string
	}

	connectivityPolicy := platform.ConnectivityOffline
	claimName := "central-claim-name"
	scannerComponentPolicy := platform.ScannerComponentEnabled
	scannerAutoScalingPolicy := platform.ScannerAutoScalingEnabled
	monitoringExposeEndpointEnabled := platform.ExposeEndpointEnabled
	monitoringExposeEndpointDisabled := platform.ExposeEndpointDisabled
	telemetryEndpoint := "endpoint"
	telemetryKey := "key"
	telemetryDisabledKey := map[string]interface{}{
		"enabled": false,
		"storage": map[string]interface{}{"key": phonehome.DisabledKey}}
	dirtyVersion := "1.2.3-dirty"
	releaseVersion := "1.2.3"

	truth := true
	falsity := false
	lbPort := int32(12345)
	lbIP := "1.1.1.1"
	nodePortPort := int32(23456)
	scannerReplicas := int32(7)
	scannerMinReplicas := int32(6)
	scannerMaxReplicas := int32(8)
	defaultPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "stackrox",
			Name:      extensions.DefaultCentralPVCName,
		},
	}

	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"empty spec": {
			args: args{
				c: platform.Central{
					Spec: platform.CentralSpec{},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{defaultPvc},
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"createClaim": false,
						},
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
			},
		},

		"empty spec no pvc": {
			args: args{
				c: platform.Central{
					Spec: platform.CentralSpec{},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
				},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"none": true,
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},

		"pvc namespace not match": {
			args: args{
				c: platform.Central{
					Spec: platform.CentralSpec{},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: extensions.DefaultCentralPVCName}}},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"none": true,
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},

		"everything and the kitchen sink": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.CentralSpec{
						ImagePullSecrets: []platform.LocalSecretReference{
							{Name: "image-pull-secrets-secret1"},
							{Name: "image-pull-secrets-secret2"},
						},
						Egress: &platform.Egress{
							ConnectivityPolicy: &connectivityPolicy,
						},
						TLS: &platform.TLSConfig{
							AdditionalCAs: []platform.AdditionalCA{
								{Name: "ca1-name", Content: "ca1-content"},
								{Name: "ca2-name", Content: "ca2-content"},
							},
						},
						Monitoring: &platform.GlobalMonitoring{
							OpenShiftMonitoring: &platform.OpenShiftMonitoring{
								Enabled: true,
							},
						},
						Central: &platform.CentralComponentSpec{
							DeploymentSpec: platform.DeploymentSpec{
								NodeSelector: map[string]string{
									"central-node-selector-label1": "central-node-selector-value1",
									"central-node-selector-label2": "central-node-selector-value2",
								},
								Tolerations: []*corev1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: corev1.TolerationOpExists},
								},
								Resources: &corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("10"),
										corev1.ResourceMemory: resource.MustParse("20"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30"),
										corev1.ResourceMemory: resource.MustParse("40"),
									},
								},
							},
							DefaultTLSSecret: &platform.LocalSecretReference{
								Name: "my-default-tls-secret",
							},
							Monitoring: &platform.Monitoring{
								ExposeEndpoint: &monitoringExposeEndpointEnabled,
							},
							Persistence: &platform.Persistence{
								HostPath: &platform.HostPathSpec{
									Path: pointer.String("/central/host/path"),
								},
								PersistentVolumeClaim: &platform.PersistentVolumeClaim{
									ClaimName: &claimName,
								},
							},
							Exposure: &platform.Exposure{
								LoadBalancer: &platform.ExposureLoadBalancer{
									Enabled: &truth,
									Port:    &lbPort,
									IP:      &lbIP,
								},
								NodePort: &platform.ExposureNodePort{
									Enabled: &truth,
									Port:    &nodePortPort,
								},
								Route: &platform.ExposureRoute{
									Enabled: &truth,
								},
							},
							Telemetry: &platform.Telemetry{
								Enabled: &truth,
								Storage: &platform.TelemetryStorage{
									Endpoint: &telemetryEndpoint,
									Key:      &telemetryKey,
								},
							},
							DeclarativeConfiguration: &platform.DeclarativeConfiguration{
								ConfigMaps: []platform.LocalConfigMapReference{
									{
										Name: "config-map-1",
									},
									{
										Name: "config-map-2",
									},
								},
								Secrets: []platform.LocalSecretReference{
									{
										Name: "secret-1",
									},
									{
										Name: "secret-2",
									},
								},
							},
							NotifierSecretsEncryption: &platform.NotifierSecretsEncryption{
								Enabled: pointer.Bool(true),
							},
						},
						Scanner: &platform.ScannerComponentSpec{
							ScannerComponent: &scannerComponentPolicy,
							Analyzer: &platform.ScannerAnalyzerComponent{
								Scaling: &platform.ScannerAnalyzerScaling{
									AutoScaling: &scannerAutoScalingPolicy,
									Replicas:    &scannerReplicas,
									MinReplicas: &scannerMinReplicas,
									MaxReplicas: &scannerMaxReplicas,
								},
								DeploymentSpec: platform.DeploymentSpec{
									NodeSelector: map[string]string{
										"scanner-node-selector-label1": "scanner-node-selector-value1",
										"scanner-node-selector-label2": "scanner-node-selector-value2",
									},
									Tolerations: []*corev1.Toleration{
										{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
										{Key: "node-role.kubernetes.io/infra", Value: "", Operator: corev1.TolerationOpExists},
									},
									Resources: &corev1.ResourceRequirements{
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("50"),
											corev1.ResourceMemory: resource.MustParse("60"),
										},
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("70"),
											corev1.ResourceMemory: resource.MustParse("80"),
										},
									},
								},
							},
							DB: &platform.DeploymentSpec{
								NodeSelector: map[string]string{
									"scanner-db-node-selector-label1": "scanner-db-node-selector-value1",
									"scanner-db-node-selector-label2": "scanner-db-node-selector-value2",
								},
								Tolerations: []*corev1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: corev1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: corev1.TolerationOpExists},
								},
								Resources: &corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("90"),
										corev1.ResourceMemory: resource.MustParse("100"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("110"),
										corev1.ResourceMemory: resource.MustParse("120"),
									},
								},
							},
							Monitoring: &platform.Monitoring{
								ExposeEndpoint: &monitoringExposeEndpointEnabled,
							},
						},
						Customize: &platform.CustomizeSpec{
							Labels: map[string]string{
								"customize-label1": "customize-label1-value",
								"customize-label2": "customize-label2-value",
							},
							Annotations: map[string]string{
								"customize-annotation1": "customize-annotation1-value",
								"customize-annotation2": "customize-annotation2-value",
							},
							EnvVars: []corev1.EnvVar{
								{
									Name:  "customize-env-var1",
									Value: "customize-env-var1-value",
								},
								{
									Name:  "customize-env-var2",
									Value: "customize-env-var2-value",
								},
							},
						},
					},
				},
				clientSet: fake.NewSimpleClientset(
					makeSecret("central-tls-spec-secret",
						map[string]string{
							"key":  "central-tls-spec-secret-key-content",
							"cert": "central-tls-spec-secret-cert-content",
						}),
					makeSecret("scanner-tls-spec-secret",
						map[string]string{
							"key":  "scanner-tls-spec-secret-key-content",
							"cert": "scanner-tls-spec-secret-cert-content",
						}),
					makeSecret("scanner-db-tls-spec-secret",
						map[string]string{
							"key":  "scanner-db-tls-spec-secret-key-content",
							"cert": "scanner-db-tls-spec-secret-cert-content",
						}),
				),
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"additionalCAs": map[string]interface{}{
					"ca1-name": "ca1-content",
					"ca2-name": "ca2-content",
				},
				"central": map[string]interface{}{
					"defaultTLS": map[string]interface{}{
						"reference": "my-default-tls-secret",
					},
					"exposure": map[string]interface{}{
						"loadBalancer": map[string]interface{}{
							"enabled": true,
							"ip":      "1.1.1.1",
							"port":    int32(12345),
						},
						"nodePort": map[string]interface{}{
							"enabled": true,
							"port":    int32(23456),
						},
						"route": map[string]interface{}{
							"enabled": true,
						},
					},
					"nodeSelector": map[string]string{
						"central-node-selector-label1": "central-node-selector-value1",
						"central-node-selector-label2": "central-node-selector-value2",
					},
					"tolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
					"exposeMonitoring": true,
					"persistence": map[string]interface{}{
						"hostPath": "/central/host/path",
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "10",
							"memory": "20",
						},
						"requests": map[string]interface{}{
							"cpu":    "30",
							"memory": "40",
						},
					},
					"telemetry": map[string]interface{}{
						"enabled": true,
						"storage": map[string]interface{}{
							"endpoint": "endpoint",
							"key":      "key",
						},
					},
					"declarativeConfiguration": map[string]interface{}{
						"mounts": map[string]interface{}{
							"configMaps": []string{
								"config-map-1",
								"config-map-2",
							},
							"secrets": []string{
								"secret-1",
								"secret-2",
							},
						},
					},
					"notifierSecretsEncryption": map[string]interface{}{
						"enabled": true,
					},
				},
				"env": map[string]interface{}{
					"offlineMode": true,
				},
				"imagePullSecrets": map[string]interface{}{
					"useExisting": []string{
						"image-pull-secrets-secret1",
						"image-pull-secrets-secret2",
					},
				},
				"customize": map[string]interface{}{
					"annotations": map[string]string{
						"customize-annotation1": "customize-annotation1-value",
						"customize-annotation2": "customize-annotation2-value",
					},
					"labels": map[string]string{
						"customize-label1": "customize-label1-value",
						"customize-label2": "customize-label2-value",
					},
					"envVars": map[string]interface{}{
						"customize-env-var1": map[string]interface{}{
							"value": "customize-env-var1-value",
						},
						"customize-env-var2": map[string]interface{}{
							"value": "customize-env-var2-value",
						},
					},
				},
				"scanner": map[string]interface{}{
					"disable":  false,
					"replicas": int32(7),
					"autoscaling": map[string]interface{}{
						"disable":     false,
						"minReplicas": int32(6),
						"maxReplicas": int32(8),
					},
					"nodeSelector": map[string]string{
						"scanner-node-selector-label1": "scanner-node-selector-value1",
						"scanner-node-selector-label2": "scanner-node-selector-value2",
					},
					"tolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
					"dbNodeSelector": map[string]string{
						"scanner-db-node-selector-label1": "scanner-db-node-selector-value1",
						"scanner-db-node-selector-label2": "scanner-db-node-selector-value2",
					},
					"dbTolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "50",
							"memory": "60",
						},
						"requests": map[string]interface{}{
							"cpu":    "70",
							"memory": "80",
						},
					},
					"dbResources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "90",
							"memory": "100",
						},
						"requests": map[string]interface{}{
							"cpu":    "110",
							"memory": "120",
						},
					},
					"exposeMonitoring": true,
				},
				"system": map[string]interface{}{
					"createSCCs": true,
				},
			},
		},

		"with configured PVC": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Persistence: &platform.Persistence{
								PersistentVolumeClaim: &platform.PersistentVolumeClaim{
									ClaimName:        pointer.String("stackrox-db-test"),
									StorageClassName: pointer.String("storage-class"),
									Size:             pointer.String("50Gi"),
								},
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "stackrox",
							Name:      "stackrox-db-test",
						},
					},
				},
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"claimName":   "stackrox-db-test",
							"createClaim": false,
						},
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
			},
		},

		"with configured pvc disabled by annotation": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{common.CentralPVCObsoleteAnnotation: "true"},
						Namespace:   "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Persistence: &platform.Persistence{
								PersistentVolumeClaim: &platform.PersistentVolumeClaim{
									ClaimName:        pointer.String("stackrox-db-test"),
									StorageClassName: pointer.String("storage-class"),
									Size:             pointer.String("50Gi"),
								},
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "stackrox",
							Name:      "stackrox-db-test",
						},
					},
				},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"none": true,
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},

		"configured PVC does not exist": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Persistence: &platform.Persistence{
								PersistentVolumeClaim: &platform.PersistentVolumeClaim{
									ClaimName:        pointer.String("stackrox-db-test"),
									StorageClassName: pointer.String("storage-class"),
									Size:             pointer.String("50Gi"),
								},
							},
						},
					},
				},
			},

			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"none": true,
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},

		"disabled monitoring endpoint": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Monitoring: &platform.Monitoring{
								ExposeEndpoint: &monitoringExposeEndpointDisabled,
							},
						},
						Scanner: &platform.ScannerComponentSpec{
							Monitoring: &platform.Monitoring{
								ExposeEndpoint: &monitoringExposeEndpointDisabled,
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{defaultPvc},
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"createClaim": false,
						},
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"scanner": map[string]interface{}{
					"exposeMonitoring": false,
				},
			},
		},

		"route with custom hostname": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Exposure: &platform.Exposure{
								Route: &platform.ExposureRoute{
									Enabled: &truth,
									Host:    pointer.String("custom-route.stackrox.io"),
								},
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{defaultPvc},
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"central": map[string]interface{}{
					"exposure": map[string]interface{}{
						"route": map[string]interface{}{
							"enabled": true,
							"host":    "custom-route.stackrox.io",
						},
					},
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"createClaim": false,
						},
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
			},
		},

		"add managed service setting": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace:   "stackrox",
						Annotations: map[string]string{managedServicesAnnotation: "true"},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{defaultPvc},
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"createClaim": false,
						},
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"env": map[string]interface{}{
					"managedServices": true,
				},
			},
		},

		"disabled telemetry": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Telemetry: &platform.Telemetry{
								Enabled: &falsity,
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{defaultPvc},
			},
			want: chartutil.Values{
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence":      map[string]interface{}{"persistentVolumeClaim": map[string]interface{}{"createClaim": false}},
					"telemetry":        telemetryDisabledKey,
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
			},
		},
		"default dev telemetry": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
				},
				pvcs:    []*corev1.PersistentVolumeClaim{defaultPvc},
				version: dirtyVersion,
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence":      map[string]interface{}{"persistentVolumeClaim": map[string]interface{}{"createClaim": false}},
					"telemetry":        telemetryDisabledKey,
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		"enabled telemetry in dev": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Telemetry: &platform.Telemetry{
								Enabled: &truth,
								Storage: &platform.TelemetryStorage{
									Key:      &telemetryKey,
									Endpoint: &telemetryEndpoint,
								},
							},
						},
					},
				},
				pvcs:    []*corev1.PersistentVolumeClaim{defaultPvc},
				version: dirtyVersion,
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence":      map[string]interface{}{"persistentVolumeClaim": map[string]interface{}{"createClaim": false}},
					"telemetry": map[string]interface{}{
						"enabled": true,
						"storage": map[string]interface{}{
							"endpoint": "endpoint",
							"key":      "key",
						},
					},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		"enabled telemetry no key": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Telemetry: &platform.Telemetry{
								Enabled: &truth,
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{defaultPvc},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"persistence":      map[string]interface{}{"persistentVolumeClaim": map[string]interface{}{"createClaim": false}},
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"createClaim": false,
							},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if !buildinfo.ReleaseBuild || buildinfo.TestBuild {
				wantCentral := tt.want["central"].(map[string]any)
				if _, ok := wantCentral["telemetry"]; !ok {
					wantCentral["telemetry"] = telemetryDisabledKey
				}
			}
			if tt.args.version == "" {
				testutils.SetMainVersion(t, releaseVersion)
			} else {
				testutils.SetMainVersion(t, tt.args.version)
			}

			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")
			var allExisting []ctrlClient.Object
			for _, existingPVC := range tt.args.pvcs {
				allExisting = append(allExisting, existingPVC)
			}
			client := fkClient.NewClientBuilder().WithObjects(allExisting...).Build()
			translator := New(client)

			got, err := translator.translate(context.Background(), tt.args.c)
			assert.NoError(t, err)

			assert.Equal(t, wantAsValues, got)
		})
	}
}

func makeSecret(name string, stringData map[string]string) *corev1.Secret {
	data := map[string][]byte{}
	for key, val := range stringData {
		data[key] = []byte(val)
	}
	return &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: "stackrox"},
		Data:       data,
	}
}
