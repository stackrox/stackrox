package certrefresh

import (
	"context"
	"crypto/x509"
	"testing"

	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHandleCABundleConfigMapUpdate(t *testing.T) {
	t.Run("with valid sensor deployment setup", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "test-namespace")
		t.Setenv("POD_NAME", "test-pod")

		k8sClient := fake.NewSimpleClientset(createTestPod("test-pod", "test-namespace", "sensor-rs"),
			createTestReplicaSet("sensor-rs", "test-namespace", "sensor"))
		centralCAs := []*x509.Certificate{testutils.IssueSelfSignedCert(t, "Primary CA").Leaf,
			testutils.IssueSelfSignedCert(t, "Secondary CA").Leaf}

		handleCABundleConfigMapUpdate(context.Background(), centralCAs, k8sClient)

		configMap, err := k8sClient.CoreV1().ConfigMaps("test-namespace").Get(
			context.Background(), pkgKubernetes.TLSCABundleConfigMapName, metav1.GetOptions{})
		require.NoError(t, err, "ConfigMap should have been created successfully")

		// Verify ownerRef points to the Sensor deployment
		require.Len(t, configMap.OwnerReferences, 1, "ConfigMap should have one owner reference")
		ownerRef := configMap.OwnerReferences[0]
		assert.Equal(t, "Deployment", ownerRef.Kind)
		assert.Equal(t, "sensor", ownerRef.Name)
		assert.Equal(t, "deployment-uid", string(ownerRef.UID))
	})
}
