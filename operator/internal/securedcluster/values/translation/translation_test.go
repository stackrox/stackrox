package translation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/jeremywohl/flatten"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/images"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	testingUtils "github.com/stackrox/rox/operator/internal/values/testing"
	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TranslationTestSuite struct {
	suite.Suite
}

func TestTranslation(t *testing.T) {
	suite.Run(t, new(TranslationTestSuite))
}

func (s *TranslationTestSuite) TestImageOverrides() {
	s.T().Setenv(images.ScannerSlim.EnvVar(), "stackrox/scanner:1.0.0")
	s.T().Setenv(images.ScannerSlimDB.EnvVar(), "stackrox/scanner-db:1.0.0")
	s.T().Setenv(images.ScannerV4DB.EnvVar(), "stackrox/scanner-v4-db:1.0.0")
	s.T().Setenv(images.ScannerV4.EnvVar(), "stackrox/scanner-v4:1.0.0")

	obj := platform.SecuredCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "stackrox",
			Name:      "secured-cluster",
		},
	}
	u, err := toUnstructured(obj)
	s.Require().NoError(err)

	fc := newDefaultFakeClient(s.T())
	translator := Translator{client: fc, direct: fc}

	vals, err := translator.Translate(context.Background(), u)
	s.Require().NoError(err)

	scannerImage, err := vals.PathValue("image.scanner.fullRef")
	s.Require().NoError(err)
	s.Equal("stackrox/scanner:1.0.0", scannerImage)

	scannerDbImage, err := vals.PathValue("image.scannerDb.fullRef")
	s.Require().NoError(err)
	s.Equal("stackrox/scanner-db:1.0.0", scannerDbImage)

	scannerV4DbImage, err := vals.PathValue("image.scannerV4DB.fullRef")
	s.Require().NoError(err)
	s.Equal("stackrox/scanner-v4-db:1.0.0", scannerV4DbImage)

	scannerV4Image, err := vals.PathValue("image.scannerV4.fullRef")
	s.Require().NoError(err)
	s.Equal("stackrox/scanner-v4:1.0.0", scannerV4Image)
}

func TestReadBaseValues(t *testing.T) {
	_, err := chartutil.ReadValues(baseValuesYAML)
	assert.NoError(t, err)
}

func TestTranslateShouldCreateConfigFingerprint(t *testing.T) {
	sc := platform.SecuredCluster{
		Spec: platform.SecuredClusterSpec{
			ClusterName: "my-cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "stackrox",
			Name:      "my-secured-cluster",
		},
	}

	u, err := toUnstructured(sc)
	require.NoError(t, err)

	fc := newDefaultFakeClient(t)
	translator := Translator{client: fc, direct: fc}
	vals, err := translator.Translate(context.Background(), u)
	require.NoError(t, err)

	testingUtils.AssertPathValueMatches(t, vals, regexp.MustCompile("[0-9a-f]{32}"), "meta.configFingerprintOverride")
}

func (s *TranslationTestSuite) TestTranslate() {
	t := s.T()

	type args struct {
		client ctrlClient.Client
		sc     platform.SecuredCluster
	}

	scannerComponentPolicy := platform.LocalScannerComponentAutoSense
	scannerAutoScalingPolicy := platform.ScannerAutoScalingEnabled
	motoringEnabled := platform.ExposeEndpointEnabled
	scannerReplicas := int32(7)
	scannerMinReplicas := int32(6)
	scannerMaxReplicas := int32(8)

	// TODO(ROX-7647): Add sensor, collector and compliance tests
	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"minimal spec": {
			args: args{
				client: newDefaultFakeClient(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": false,
				},
				"scannerV4": map[string]interface{}{
					"disable": true,
				},
				"sensor": map[string]interface{}{
					"localImageScanning": map[string]string{
						"enabled": "true",
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		"local scanner autosense suppression": {
			args: args{
				client: newDefaultFakeClientWithCentral(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
						Scanner: &platform.LocalScannerComponentSpec{
							ScannerComponent: platform.LocalScannerComponentDisabled.Pointer(),
						},
						ScannerV4: &platform.LocalScannerV4ComponentSpec{
							ScannerComponent: platform.LocalScannerV4ComponentDisabled.Pointer(),
						},
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": true,
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"scannerV4": map[string]interface{}{
					"disable": true,
				},
			},
		},
		"scannerV4.db.persistence.none for Scanner V4 db without default StorageClass": {
			args: args{
				// no default storage class in this fake client, so we expect to default to scannerV4.db.persistence.none
				client: newDefaultFakeClientWithoutStorageClass(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": false,
				},
				"sensor": map[string]interface{}{
					"localImageScanning": map[string]interface{}{
						"enabled": "true",
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"scannerV4": map[string]interface{}{
					"disable": true,
				},
			},
		},
		"local scanner autosense no suppression": {
			args: args{
				client: newDefaultFakeClient(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": false,
				},
				"scannerV4": map[string]interface{}{
					"disable": true,
				},
				"sensor": map[string]interface{}{
					"localImageScanning": map[string]string{
						"enabled": "true",
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		"complete spec": {
			args: args{
				client: newDefaultFakeClient(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-secured-cluster",
						Namespace: "stackrox",
					},
					Spec: platform.SecuredClusterSpec{
						ClusterName:     "test-cluster",
						CentralEndpoint: "central.test:443",
						Sensor: &platform.SensorComponentSpec{
							DeploymentSpec: platform.DeploymentSpec{
								Tolerations: []*v1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
								},
								HostAliases: []v1.HostAlias{
									{
										IP:        "127.0.0.1",
										Hostnames: []string{"sensor.com"},
									},
								},
							},
						},
						AdmissionControl: &platform.AdmissionControlComponentSpec{
							ListenOnCreates:      pointer.Bool(true),
							ListenOnUpdates:      pointer.Bool(false),
							ListenOnEvents:       pointer.Bool(true),
							ContactImageScanners: platform.ScanIfMissing.Pointer(),
							TimeoutSeconds:       pointer.Int32(4),
							Bypass:               platform.BypassBreakGlassAnnotation.Pointer(),
							DeploymentSpec: platform.DeploymentSpec{
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1502m"),
										v1.ResourceMemory: resource.MustParse("1002Mi"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1501m"),
										v1.ResourceMemory: resource.MustParse("1001Mi"),
									},
								},
								NodeSelector: map[string]string{
									"admission-ctrl-node-selector1": "admission-ctrl-node-selector-val1",
									"admission-ctrl-node-selector2": "admission-ctrl-node-selector-val2",
								},
								Tolerations: []*v1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
								},
								HostAliases: []v1.HostAlias{
									{
										IP:        "127.0.0.1",
										Hostnames: []string{"admission-control.com"},
									},
								},
							},
						},
						ClusterLabels: map[string]string{
							"my-label1": "value1",
							"my-label2": "value2",
						},
						ImagePullSecrets: []platform.LocalSecretReference{
							{Name: "image-pull-secrets-secret1"},
							{Name: "image-pull-secrets-secret2"},
						},
						TLS: &platform.TLSConfig{
							AdditionalCAs: []platform.AdditionalCA{
								{Name: "ca1-name", Content: "ca1-content"},
								{Name: "ca2-name", Content: "ca2-content"},
							},
						},
						AuditLogs: &platform.AuditLogsSpec{
							Collection: platform.AuditLogsCollectionEnabled.Pointer(),
						},
						PerNode: &platform.PerNodeSpec{
							Collector: &platform.CollectorContainerSpec{
								ImageFlavor: platform.ImageFlavorRegular.Pointer(),
								Collection:  platform.CollectionCOREBPF.Pointer(),
							},
							TaintToleration: platform.TaintTolerate.Pointer(),
							Compliance: &platform.ContainerSpec{
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1504m"),
										v1.ResourceMemory: resource.MustParse("1004Mi"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1503m"),
										v1.ResourceMemory: resource.MustParse("1003Mi"),
									},
								},
							},
							HostAliases: []v1.HostAlias{
								{
									IP:        "127.0.0.1",
									Hostnames: []string{"collector.com"},
								},
							},
						},
						Monitoring: &platform.GlobalMonitoring{
							OpenShiftMonitoring: &platform.OpenShiftMonitoring{
								Enabled: true,
							},
						},
						Scanner: &platform.LocalScannerComponentSpec{
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
									Tolerations: []*v1.Toleration{
										{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
										{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
									},
									Resources: &v1.ResourceRequirements{
										Limits: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("50"),
											v1.ResourceMemory: resource.MustParse("60"),
										},
										Requests: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("70"),
											v1.ResourceMemory: resource.MustParse("80"),
										},
									},
									HostAliases: []v1.HostAlias{
										{
											IP:        "127.0.0.1",
											Hostnames: []string{"scanner.com"},
										},
									},
								},
							},
							DB: &platform.DeploymentSpec{
								NodeSelector: map[string]string{
									"scanner-db-node-selector-label1": "scanner-db-node-selector-value1",
									"scanner-db-node-selector-label2": "scanner-db-node-selector-value2",
								},
								Tolerations: []*v1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
								},
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("90"),
										v1.ResourceMemory: resource.MustParse("100"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("110"),
										v1.ResourceMemory: resource.MustParse("120"),
									},
								},
								HostAliases: []v1.HostAlias{
									{
										IP:        "127.0.0.1",
										Hostnames: []string{"scanner-db.com"},
									},
								},
							},
						},
						ScannerV4: &platform.LocalScannerV4ComponentSpec{
							ScannerComponent: platform.LocalScannerV4ComponentAutoSense.Pointer(),
							Indexer: &platform.ScannerV4Component{
								Scaling: &platform.ScannerComponentScaling{
									AutoScaling: &scannerAutoScalingPolicy,
									Replicas:    &scannerReplicas,
									MinReplicas: &scannerMinReplicas,
									MaxReplicas: &scannerMaxReplicas,
								},
								DeploymentSpec: platform.DeploymentSpec{
									NodeSelector: map[string]string{
										"scanner-v4-indexer-node-selector-label1": "scanner-v4-indexer-node-selector-value1",
										"scanner-v4-indexer-node-selector-label2": "scanner-v4-indexer-node-selector-value2",
									},
									Tolerations: []*v1.Toleration{
										{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
										{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
									},
									Resources: &v1.ResourceRequirements{
										Limits: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("110"),
											v1.ResourceMemory: resource.MustParse("120"),
										},
										Requests: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("100"),
											v1.ResourceMemory: resource.MustParse("110"),
										},
									},
									HostAliases: []v1.HostAlias{
										{
											IP:        "127.0.0.1",
											Hostnames: []string{"scanner-v4-indexer.com"},
										},
									},
								},
							},
							DB: &platform.ScannerV4DB{
								Persistence: &platform.ScannerV4Persistence{
									PersistentVolumeClaim: &platform.ScannerV4PersistentVolumeClaim{
										ClaimName:        pointer.String("scanner-v4-db-pvc"),
										StorageClassName: pointer.String("test-sc1"),
									},
								},
								DeploymentSpec: platform.DeploymentSpec{
									NodeSelector: map[string]string{
										"scanner-v4-db-node-selector-label1": "scanner-v4-db-node-selector-value1",
										"scanner-v4-db-node-selector-label2": "scanner-v4-db-node-selector-value2",
									},
									Tolerations: []*v1.Toleration{
										{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
										{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
									},
									Resources: &v1.ResourceRequirements{
										Limits: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("110"),
											v1.ResourceMemory: resource.MustParse("120"),
										},
										Requests: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("100"),
											v1.ResourceMemory: resource.MustParse("110"),
										},
									},
									HostAliases: []v1.HostAlias{
										{
											IP:        "127.0.0.1",
											Hostnames: []string{"scanner-v4-db.com"},
										},
									},
								},
							},
							Monitoring: &platform.Monitoring{
								ExposeEndpoint: &motoringEnabled,
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
							EnvVars: []v1.EnvVar{
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
						RegistryOverride: "my.registry.override.com",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":     "test-cluster",
				"centralEndpoint": "central.test:443",
				"clusterLabels": map[string]interface{}{
					"my-label1": "value1",
					"my-label2": "value2",
				},
				"imagePullSecrets": map[string]interface{}{
					"useExisting": []string{
						"image-pull-secrets-secret1",
						"image-pull-secrets-secret2",
					},
				},
				"additionalCAs": map[string]interface{}{
					"ca1-name": "ca1-content",
					"ca2-name": "ca2-content",
				},
				"sensor": map[string]interface{}{
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
					"localImageScanning": map[string]string{
						"enabled": "true",
					},
					"hostAliases": []map[string]interface{}{
						{
							"ip":        "127.0.0.1",
							"hostnames": []string{"sensor.com"},
						},
					},
				},
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": false,
						"scanInline":       true,
						"disableBypass":    false,
						"timeout":          4,
					},
					"listenOnCreates": true,
					"listenOnUpdates": false,
					"listenOnEvents":  true,
					"nodeSelector": map[string]interface{}{
						"admission-ctrl-node-selector1": "admission-ctrl-node-selector-val1",
						"admission-ctrl-node-selector2": "admission-ctrl-node-selector-val2",
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "1502m",
							"memory": "1002Mi",
						}, "requests": map[string]interface{}{
							"cpu":    "1501m",
							"memory": "1001Mi",
						},
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
					"hostAliases": []map[string]interface{}{
						{
							"ip":        "127.0.0.1",
							"hostnames": []string{"admission-control.com"},
						},
					},
				},
				"auditLogs": map[string]interface{}{
					"disableCollection": false,
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
					"hostAliases": []map[string]interface{}{
						{
							"ip":        "127.0.0.1",
							"hostnames": []string{"scanner.com"},
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
					"dbHostAliases": []map[string]interface{}{
						{
							"ip":        "127.0.0.1",
							"hostnames": []string{"scanner-db.com"},
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
								"cpu":    "110",
								"memory": "120",
							},
							"requests": map[string]interface{}{
								"cpu":    "100",
								"memory": "110",
							},
						},
						"nodeSelector": map[string]string{
							"scanner-v4-indexer-node-selector-label1": "scanner-v4-indexer-node-selector-value1",
							"scanner-v4-indexer-node-selector-label2": "scanner-v4-indexer-node-selector-value2",
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
						"hostAliases": []map[string]interface{}{
							{
								"ip":        "127.0.0.1",
								"hostnames": []string{"scanner-v4-indexer.com"},
							},
						},
					},
					"db": map[string]interface{}{
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"cpu":    "110",
								"memory": "120",
							},
							"requests": map[string]interface{}{
								"cpu":    "100",
								"memory": "110",
							},
						},
						"nodeSelector": map[string]string{
							"scanner-v4-db-node-selector-label1": "scanner-v4-db-node-selector-value1",
							"scanner-v4-db-node-selector-label2": "scanner-v4-db-node-selector-value2",
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
							"persistentVolumeClaim": map[string]interface{}{
								"claimName":    "scanner-v4-db-pvc",
								"createClaim":  true,
								"storageClass": "test-sc1",
							},
						},
						"hostAliases": []map[string]interface{}{
							{
								"ip":        "127.0.0.1",
								"hostnames": []string{"scanner-v4-db.com"},
							},
						},
					},
					"exposeMonitoring": true,
				},
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
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
				"collector": map[string]interface{}{
					"forceCollectionMethod":   true,
					"collectionMethod":        "CORE_BPF",
					"disableTaintTolerations": false,
					"complianceResources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "1504m",
							"memory": "1004Mi",
						}, "requests": map[string]interface{}{
							"cpu":    "1503m",
							"memory": "1003Mi",
						},
					},
					"hostAliases": []map[string]interface{}{
						{
							"ip":        "127.0.0.1",
							"hostnames": []string{"collector.com"},
						},
					},
				},
				"monitoring": map[string]interface{}{
					"openshift": map[string]interface{}{
						"enabled": true,
					},
				},
				"registryOverride": "my.registry.override.com",
			},
		},
		"translate EBPF to CORE_BPF": {
			args: args{
				client: newDefaultFakeClient(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
						PerNode: &platform.PerNodeSpec{
							Collector: &platform.CollectorContainerSpec{
								ImageFlavor: platform.ImageFlavorRegular.Pointer(),
								Collection:  platform.CollectionEBPF.Pointer(),
							},
						},
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"collector": map[string]interface{}{
					"forceCollectionMethod": true,
					"collectionMethod":      "CORE_BPF",
				},
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": false,
				},
				"scannerV4": map[string]interface{}{
					"disable": true,
				},
				"sensor": map[string]interface{}{
					"localImageScanning": map[string]string{
						"enabled": "true",
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
			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			translator := Translator{client: tt.args.client, direct: tt.args.client}
			got, err := translator.translate(context.Background(), tt.args.sc)
			require.NoError(t, err)

			// Remove config fingerprint as it changes as the test case changes
			_, err = got.PathValue("meta.configFingerprintOverride")
			require.NoError(t, err)
			delete(got["meta"].(map[string]interface{}), "configFingerprintOverride")
			if len(got["meta"].(map[string]interface{})) == 0 {
				delete(got, "meta")
			}

			assert.Equal(t, wantAsValues, got)
		})
	}
}

func TestTranslatePartialMatch(t *testing.T) {
	type args struct {
		sc platform.SecuredCluster
	}

	networkPoliciesEnabled := platform.NetworkPoliciesEnabled
	networkPoliciesDisabled := platform.NetworkPoliciesDisabled

	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"unset network": {
			args: args{
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec:       platform.SecuredClusterSpec{},
				},
			},
			want: chartutil.Values{
				"network":                       nil,
				"network.enableNetworkPolicies": nil,
			},
		},
		"unset network policies": {
			args: args{
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
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
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
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
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			client := newDefaultFakeClientWithCentral(t) // Provide default objects and central for detection
			translator := New(client, client)
			got, err := translator.translate(context.Background(), tt.args.sc)
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

func toUnstructured(sc platform.SecuredCluster) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&sc)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

var defaultObjects = []ctrlClient.Object{
	createSecret(sensorTLSSecretName),
	createSecret(collectorTLSSecretName),
	createSecret(admissionControlTLSSecretName),
	testutils.ValidClusterVersion,
}

var defaultStorageClasses = []ctrlClient.Object{
	&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{
		Name: "test-sc1",
	}},
	&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{
		Name: "test-sc2",
		Annotations: map[string]string{
			translation.DefaultStorageClassAnnotationKey: "true",
		},
	}},
}

func newDefaultFakeClient(t *testing.T) ctrlClient.Client {
	objects := append(defaultObjects, defaultStorageClasses...)
	return testutils.NewFakeClientBuilder(t, objects...).Build()
}

func newDefaultFakeClientWithCentral(t *testing.T) ctrlClient.Client {
	objects := append(defaultObjects, defaultStorageClasses...)
	objects = append(objects, &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-central",
			Namespace: "stackrox",
		},
	})
	return testutils.NewFakeClientBuilder(t, objects...).Build()
}

func newDefaultFakeClientWithoutStorageClass(t *testing.T) ctrlClient.Client {
	return testutils.NewFakeClientBuilder(t, defaultObjects...).Build()
}

func createSecret(name string) *v1.Secret {
	serviceName := strings.TrimSuffix(name, "-tls")

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "stackrox",
		},
		Data: map[string][]byte{
			"ca.pem":                                []byte(`ca central content`),
			fmt.Sprintf("%s-key.pem", serviceName):  []byte(`key content`),
			fmt.Sprintf("%s-cert.pem", serviceName): []byte(`cert content`),
		},
	}
}
