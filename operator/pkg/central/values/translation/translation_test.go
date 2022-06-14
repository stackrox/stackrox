package translation

import (
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
)

func TestReadBaseValues(t *testing.T) {
	_, err := chartutil.ReadValues(baseValuesYAML)
	assert.NoError(t, err)
}

func TestTranslate(t *testing.T) {
	type args struct {
		clientSet kubernetes.Interface
		c         platform.Central
	}

	connectivityPolicy := platform.ConnectivityOffline
	claimName := "central-claim-name"
	scannerComponentPolicy := platform.ScannerComponentEnabled
	scannerAutoScalingPolicy := platform.ScannerAutoScalingEnabled

	truth := true
	lbPort := int32(12345)
	lbIP := "1.1.1.1"
	nodePortPort := int32(23456)
	scannerReplicas := int32(7)
	scannerMinReplicas := int32(6)
	scannerMaxReplicas := int32(8)

	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"empty spec": {
			args: args{
				c: platform.Central{
					Spec: platform.CentralSpec{},
				},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"createClaim": false,
						},
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
							Persistence: &platform.Persistence{
								HostPath: &platform.HostPathSpec{
									Path: pointer.StringPtr("/central/host/path"),
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
						Misc: &platform.MiscSpec{
							CreateSCCs: pointer.BoolPtr(true),
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
					"persistence": map[string]interface{}{
						"hostPath": "/central/host/path",
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
				},
				"system": map[string]interface{}{
					"createSCCs": true,
				},
			},
		},

		"with configured PVC": {
			args: args{
				c: platform.Central{
					Spec: platform.CentralSpec{
						Central: &platform.CentralComponentSpec{
							Persistence: &platform.Persistence{
								PersistentVolumeClaim: &platform.PersistentVolumeClaim{
									ClaimName:        pointer.StringPtr("stackrox-db-test"),
									StorageClassName: pointer.StringPtr("storage-class"),
									Size:             pointer.StringPtr("50Gi"),
								},
							},
						},
					},
				},
			},
			want: chartutil.Values{
				"central": map[string]interface{}{
					"persistence": map[string]interface{}{
						"persistentVolumeClaim": map[string]interface{}{
							"claimName":   "stackrox-db-test",
							"createClaim": false,
						},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			got, err := translate(tt.args.c)
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
