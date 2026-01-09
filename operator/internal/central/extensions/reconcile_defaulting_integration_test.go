//go:build integration

package extensions

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

var (
	testEnv    *envtest.Environment
	cfg        *rest.Config
	k8sClient  ctrlClient.Client
	testScheme *runtime.Scheme
)

func TestDefaultingIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Central Defaulting Extension Integration Tests")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:        []string{"../../../config/crd/bases"},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: false, // set to true to see kube-apiserver and etcd logs
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	testScheme = runtime.NewScheme()
	err = platform.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())
	err = scheme.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())
	err = apiextv1.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = ctrlClient.New(cfg, ctrlClient.Options{Scheme: testScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})

var _ = Describe("FeatureDefaultingExtension", func() {
	var (
		ctx       context.Context
		namespace *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create namespace.
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testutils.TestNamespace,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())
	})

	// This test requires a real API server, not just a fake client.
	Describe("spec modification", func() {
		It("should not modify the spec during reconciliation", func() {
			// Decode and create the Central object.
			centralYAML := `
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
  namespace: ` + testutils.TestNamespace + `
spec:
  central:
    db:
      resources:
        limits:
          memory: 4Gi
          cpu: 1 # We intentionally use an integer here to test that spec is not modified to a string.
`
			unstructuredCentral := &unstructured.Unstructured{}
			err := yaml.Unmarshal([]byte(centralYAML), unstructuredCentral)
			Expect(err).NotTo(HaveOccurred())
			gvk := unstructuredCentral.GroupVersionKind()
			objectKey := ctrlClient.ObjectKeyFromObject(unstructuredCentral)
			err = k8sClient.Create(ctx, unstructuredCentral)
			Expect(err).NotTo(HaveOccurred())

			// Refetch the object to ensure we have the latest version as stored on the cluster.
			unstructuredCentral = &unstructured.Unstructured{}
			unstructuredCentral.SetGroupVersionKind(gvk)
			err = k8sClient.Get(ctx, objectKey, unstructuredCentral)
			Expect(err).NotTo(HaveOccurred())

			// Copy before executing extension.
			unstructuredCentralBefore := unstructuredCentral.DeepCopy()

			// Execute the extension.
			err = reconcileFeatureDefaults(ctx, k8sClient, unstructuredCentral, logr.Discard())
			Expect(err).NotTo(HaveOccurred())

			// Check if spec was modified.
			specEqual := reflect.DeepEqual(unstructuredCentralBefore.Object["spec"], unstructuredCentral.Object["spec"])

			if !specEqual {
				centralSpecBefore, err := yaml.Marshal(unstructuredCentralBefore.Object["spec"])
				Expect(err).NotTo(HaveOccurred())
				centralSpecAfter, err := yaml.Marshal(unstructuredCentral.Object["spec"])
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Println("spec before defaulting extension:")
				GinkgoWriter.Println("=================================")
				GinkgoWriter.Printf("%s\n", centralSpecBefore)
				GinkgoWriter.Println("")
				GinkgoWriter.Println("spec after defaulting extension:")
				GinkgoWriter.Println("=================================")
				GinkgoWriter.Printf("%s\n", centralSpecAfter)
			}

			Expect(specEqual).To(BeTrue(), "spec of in-memory object was modified by defaulting extension")
		})
	})
})
