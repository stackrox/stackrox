package translation

import (
	"context"
	"testing"

	"github.com/jeremywohl/flatten"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/common"
	"github.com/stackrox/rox/operator/internal/central/extensions"
	"github.com/stackrox/rox/operator/internal/values/translation"
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
	scannerComponentPolicy := platform.ScannerComponentEnabled
	scannerAutoScalingPolicy := platform.ScannerAutoScalingEnabled
	scannerV4ComponentDefault := platform.ScannerV4ComponentDefault
	monitoringExposeEndpointEnabled := platform.ExposeEndpointEnabled
	monitoringExposeEndpointDisabled := platform.ExposeEndpointDisabled
	configAsCodeComponentEnabled := platform.ConfigAsCodeComponentEnabled
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
			Name:      extensions.DefaultCentralDBPVCName,
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
				pvcs: []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: extensions.DefaultCentralDBPVCName}}},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
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
								Scaling: &platform.ScannerComponentScaling{
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
						ScannerV4: &platform.ScannerV4Spec{
							ScannerComponent: &scannerV4ComponentDefault,
							Indexer: &platform.ScannerV4Component{
								Scaling: &platform.ScannerComponentScaling{
									AutoScaling: &scannerAutoScalingPolicy,
									Replicas:    &scannerReplicas,
									MinReplicas: &scannerMinReplicas,
									MaxReplicas: &scannerMaxReplicas,
								},
								DeploymentSpec: platform.DeploymentSpec{
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
									NodeSelector: map[string]string{
										"scanner-v4-indexer-node-selector": "test",
									},
									Tolerations: []*corev1.Toleration{
										{Key: "scanner-v4-indexer-toleration", Operator: corev1.TolerationOpExists},
									},
								},
							},
							Matcher: &platform.ScannerV4Component{
								Scaling: &platform.ScannerComponentScaling{
									AutoScaling: &scannerAutoScalingPolicy,
									Replicas:    &scannerReplicas,
									MinReplicas: &scannerMinReplicas,
									MaxReplicas: &scannerMaxReplicas,
								},
								DeploymentSpec: platform.DeploymentSpec{
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
									NodeSelector: map[string]string{
										"scanner-v4-matcher-node-selector": "test",
									},
									Tolerations: []*corev1.Toleration{
										{Key: "scanner-v4-matcher-toleration", Operator: corev1.TolerationOpExists},
									},
								},
							},
							DB: &platform.ScannerV4DB{
								Persistence: &platform.ScannerV4Persistence{
									PersistentVolumeClaim: &platform.ScannerV4PersistentVolumeClaim{
										ClaimName: pointer.String("scanner-v4-db-pvc"),
									},
								},
								DeploymentSpec: platform.DeploymentSpec{
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
									NodeSelector: map[string]string{
										"scanner-v4-db-node-selector": "test",
									},
									Tolerations: []*corev1.Toleration{
										{Key: "scanner-v4-db-toleration", Operator: corev1.TolerationOpExists},
									},
								},
							},
							Monitoring: &platform.Monitoring{
								ExposeEndpoint: &monitoringExposeEndpointEnabled,
							},
						},
						ConfigAsCode: &platform.ConfigAsCodeSpec{
							ComponentPolicy: &configAsCodeComponentEnabled,
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
				"scannerV4": map[string]interface{}{
					"disable": false,
					"indexer": map[string]interface{}{
						"autoscaling": map[string]interface{}{
							"disable":     false,
							"minReplicas": int32(6),
							"maxReplicas": int32(8),
						},
						"replicas": int32(7),
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"cpu":    "90",
								"memory": "100",
							},
							"requests": map[string]interface{}{
								"cpu":    "110",
								"memory": "120",
							},
						},
						"nodeSelector": map[string]string{
							"scanner-v4-indexer-node-selector": "test",
						},
						"tolerations": []map[string]interface{}{
							{"key": "scanner-v4-indexer-toleration", "operator": "Exists"},
						},
					},
					"matcher": map[string]interface{}{
						"autoscaling": map[string]interface{}{
							"disable":     false,
							"minReplicas": int32(6),
							"maxReplicas": int32(8),
						},
						"replicas": int32(7),
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"cpu":    "90",
								"memory": "100",
							},
							"requests": map[string]interface{}{
								"cpu":    "110",
								"memory": "120",
							},
						},
						"nodeSelector": map[string]string{
							"scanner-v4-matcher-node-selector": "test",
						},
						"tolerations": []map[string]interface{}{
							{"key": "scanner-v4-matcher-toleration", "operator": "Exists"},
						},
					},
					"db": map[string]interface{}{
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"cpu":    "90",
								"memory": "100",
							},
							"requests": map[string]interface{}{
								"cpu":    "110",
								"memory": "120",
							},
						},
						"nodeSelector": map[string]string{
							"scanner-v4-db-node-selector": "test",
						},
						"tolerations": []map[string]interface{}{
							{"key": "scanner-v4-db-toleration", "operator": "Exists"},
						},
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"claimName":   "scanner-v4-db-pvc",
								"createClaim": true,
							},
						},
					},
					"exposeMonitoring": true,
				},
				"configAsCode": map[string]interface{}{
					"enabled": true,
				},
			},
		},

		"with configured DB PVC": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							DB: &platform.CentralDBSpec{
								Persistence: &platform.DBPersistence{
									PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
										ClaimName:        pointer.String("central-db-test"),
										StorageClassName: pointer.String("storage-class"),
										Size:             pointer.String("50Gi"),
									},
								},
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "stackrox",
							Name:      "central-db-test",
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
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"claimName":   "central-db-test",
								"createClaim": false,
							},
						},
					},
				},
			},
		},

		"with pvc obsolete annotation": {
			args: args{
				c: platform.Central{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{common.CentralPVCObsoleteAnnotation: "true"},
						Namespace:   "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							DB: &platform.CentralDBSpec{
								Persistence: &platform.DBPersistence{
									PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
										ClaimName:        pointer.String("central-db-test"),
										StorageClassName: pointer.String("storage-class"),
										Size:             pointer.String("50Gi"),
									},
								},
							},
						},
					},
				},
				pvcs: []*corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "stackrox",
							Name:      "central-db-test",
						},
					},
				},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"claimName":   "central-db-test",
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
							DB: &platform.CentralDBSpec{
								Persistence: &platform.DBPersistence{
									PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
										ClaimName:        pointer.String("central-db-test"),
										StorageClassName: pointer.String("storage-class"),
										Size:             pointer.String("50Gi"),
									},
								},
							},
						},
					},
				},
			},

			want: chartutil.Values{
				"central": map[string]interface{}{
					"exposeMonitoring": false,
					"db": map[string]interface{}{
						"persistence": map[string]interface{}{
							"persistentVolumeClaim": map[string]interface{}{
								"claimName":   "central-db-test",
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

func TestTranslatePartialMatch(t *testing.T) {
	type args struct {
		c platform.Central
	}

	networkPoliciesEnabled := platform.NetworkPoliciesEnabled
	networkPoliciesDisabled := platform.NetworkPoliciesDisabled

	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"unset network": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{},
				},
			},
			want: chartutil.Values{
				"network":                       nil,
				"network.enableNetworkPolicies": nil,
			},
		},
		"unset network policies": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Network: &platform.GlobalNetworkSpec{},
					},
				},
			},
			want: chartutil.Values{
				"network":                       nil,
				"network.enableNetworkPolicies": nil,
			},
		},
		"disabled network policies": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Network: &platform.GlobalNetworkSpec{
							Policies: &networkPoliciesDisabled,
						},
					},
				},
			},
			want: chartutil.Values{
				"network.enableNetworkPolicies": false,
			},
		},
		"enabled network policies": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Network: &platform.GlobalNetworkSpec{
							Policies: &networkPoliciesEnabled,
						},
					},
				},
			},
			want: chartutil.Values{
				"network.enableNetworkPolicies": true,
			},
		},
		"enabled external DB, unset connection pool": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							DB: &platform.CentralDBSpec{
								ConnectionStringOverride: pointer.String("host=fake-central.stackrox:443"),
							},
						},
					},
				},
			},
			want: chartutil.Values{
				"central.db.external":                true,
				"central.db.source.connectionString": "host=fake-central.stackrox:443",
				"central.db.source.minConns":         nil,
				"central.db.source.maxConns":         nil,
			},
		},
		"enabled external DB, set connection pool": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							DB: &platform.CentralDBSpec{},
						},
					},
				},
			},
			want: chartutil.Values{
				"central.db.external": nil,
				"central.db.source":   nil,
			},
		},
		"disabled external DB, unset connection pool": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							DB: &platform.CentralDBSpec{
								ConnectionPoolSize: &platform.DBConnectionPoolSize{
									MinConnections: pointer.Int32(20),
									MaxConnections: pointer.Int32(200),
								},
							},
						},
					},
				},
			},
			want: chartutil.Values{
				"central.db.external":                nil,
				"central.db.source.connectionString": nil,
				"central.db.source.minConns":         20,
				"central.db.source.maxConns":         200,
			},
		},
		"disabled external DB, set connection pool": {
			args: args{
				c: platform.Central{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "stackrox",
					},
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							DB: &platform.CentralDBSpec{
								ConnectionPoolSize: &platform.DBConnectionPoolSize{
									MinConnections: pointer.Int32(30),
									MaxConnections: pointer.Int32(400),
								},
							},
						},
					},
				},
			},
			want: chartutil.Values{
				"central.db.external":                nil,
				"central.db.source.connectionString": nil,
				"central.db.source.minConns":         30,
				"central.db.source.maxConns":         400,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			client := fkClient.NewClientBuilder().Build()
			translator := New(client)
			got, err := translator.translate(context.Background(), tt.args.c)
			assert.NoError(t, err)

			wantFlattened, err := flatten.Flatten(wantAsValues, "", flatten.DotStyle)
			assert.NoError(t, err)
			for key, wantValue := range wantFlattened {
				gotValue, err := got.PathValue(key)
				if wantValue == nil {
					assert.Error(t, err) // The value should not exist
				} else {
					assert.NoError(t, err)
					assert.Equal(t, wantValue, gotValue)
				}
			}
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
