package translation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	testingUtils "github.com/stackrox/rox/operator/pkg/values/testing"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestTranslate(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(createSecret(sensorTLSSecretName), createSecret(collectorTLSSecretName), createSecret(admissionControlTLSSecretName))
	sc := v1alpha1.SecuredCluster{
		Spec: v1alpha1.SecuredClusterSpec{
			ClusterName: "my-cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "stackrox",
			Name:      "stackrox-secured-cluster-services",
		},
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&sc)
	require.NoError(t, err)

	translator := Translator{clientSet: fakeClient}
	vals, err := translator.Translate(context.Background(), &unstructured.Unstructured{Object: obj})
	require.NoError(t, err)

	//TODO: Assert whole values tree to detect unwanted values (which may be added by accident)
	testingUtils.AssertNotNilPathValue(t, vals, "ca.cert")
	testingUtils.AssertEqualPathValue(t, vals, "my-cluster", "clusterName")
	testingUtils.AssertPathValueMatches(t, vals, regexp.MustCompile("[0-9a-f]{32}"), "meta.configFingerprintOverride")
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
