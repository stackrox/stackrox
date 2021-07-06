package translation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	"github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	testingUtils "github.com/stackrox/rox/operator/pkg/values/testing"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReadBaseValues(t *testing.T) {
	_, err := chartutil.ReadValues(baseValuesYAML)
	assert.NoError(t, err)
}

func TestTranslateShouldCreateConfigFingerprint(t *testing.T) {
	sc := v1alpha1.SecuredCluster{
		Spec: v1alpha1.SecuredClusterSpec{
			ClusterName: "my-cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: supportedOperandNamespace,
			Name:      supportedResourceName,
		},
	}

	u, err := toUnstructured(sc)
	require.NoError(t, err)

	translator := Translator{clientSet: newFakeClientSetWithInitBundle()}
	vals, err := translator.Translate(context.Background(), u)
	require.NoError(t, err)

	testingUtils.AssertPathValueMatches(t, vals, regexp.MustCompile("[0-9a-f]{32}"), "meta.configFingerprintOverride")
}

func TestTranslateComplete(t *testing.T) {
	type args struct {
		clientSet kubernetes.Interface
		sc        v1alpha1.SecuredCluster
	}

	//TODO: Add collector, admission-control and compliance tests
	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"SecuredCluster basic spec": {
			args: args{
				clientSet: newFakeClientSetWithInitBundle(),
				sc: v1alpha1.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      supportedResourceName,
						Namespace: supportedOperandNamespace,
					},
					Spec: v1alpha1.SecuredClusterSpec{
						ClusterName:     "test-cluster",
						CentralEndpoint: "central.test:443",
						ImagePullSecrets: []common.LocalSecretReference{
							{Name: "image-pull-secrets-secret1"},
							{Name: "image-pull-secrets-secret2"},
						},
						TLS: &common.TLSConfig{
							AdditionalCAs: []common.AdditionalCA{
								{Name: "ca1-name", Content: "ca1-content"},
								{Name: "ca2-name", Content: "ca2-content"},
							},
						},
						PerNode: &v1alpha1.PerNodeSpec{
							Collector: &v1alpha1.CollectorContainerSpec{
								ImageFlavor: v1alpha1.ImageFlavorRegular.Pointer(),
								Collection:  v1alpha1.CollectionEBPF.Pointer(),
							},
							TaintToleration: v1alpha1.TaintTolerate.Pointer(),
							Compliance: &v1alpha1.ContainerSpec{
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1500m"),
										v1.ResourceMemory: resource.MustParse("1Gi"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1500m"),
										v1.ResourceMemory: resource.MustParse("1Gi"),
									},
								},
							},
						},
						Customize: &common.CustomizeSpec{
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
					},
				},
			},
			want: chartutil.Values{
				"clusterName":     "test-cluster",
				"centralEndpoint": "central.test:443",
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
					"collectionMethod":        "EBPF",
					"disableTaintTolerations": false,
					"slimMode":                false,
					"complianceResources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "1500m",
							"memory": "1Gi",
						}, "requests": map[string]interface{}{
							"cpu":    "1500m",
							"memory": "1Gi",
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

			translator := Translator{clientSet: tt.args.clientSet}
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

func toUnstructured(sc v1alpha1.SecuredCluster) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&sc)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

func newFakeClientSetWithInitBundle() *fake.Clientset {
	return fake.NewSimpleClientset(createSecret(sensorTLSSecretName), createSecret(collectorTLSSecretName), createSecret(admissionControlTLSSecretName))
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
