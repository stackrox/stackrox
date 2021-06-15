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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
)

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
						CentralEndpoint: pointer.StringPtr("central.test:443"),
						ImagePullSecrets: []v1.LocalObjectReference{
							{Name: "image-pull-secrets-secret1"},
							{Name: "image-pull-secrets-secret2"},
						},
						TLS: &v1alpha1.TLSConfig{
							AdditionalCAs: []common.AdditionalCA{
								{Name: "ca1-name", Content: "ca1-content"},
								{Name: "ca2-name", Content: "ca2-content"},
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
				"meta": map[string]string{
					"configFingerprintOverride": "af9f1f5fcf5213073d913f3b5b92883a082543083852db9f07f2889b6effe637",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			u, err := toUnstructured(tt.args.sc)
			require.NoError(t, err)

			translator := Translator{clientSet: tt.args.clientSet}
			got, err := translator.Translate(context.Background(), u)
			require.NoError(t, err)

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
