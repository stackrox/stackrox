package reconcile

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	testEnv *envtest.Environment
	cfg     *rest.Config
	gvk     = schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "TestApp"}
)

func TestReconcileExtensions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconcile Extensions Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testEnv = &envtest.Environment{
		AttachControlPlaneOutput: false, // set to true to see kube-apiserver and etcd logs
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())

	crd := BuildTestCRD(gvk)
	_, err = envtest.InstallCRDs(cfg, envtest.CRDInstallOptions{CRDs: []*apiextv1.CustomResourceDefinition{&crd}})
	Expect(err).To(BeNil())
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})

// BuildTestCRD builds test CRD
func BuildTestCRD(gvk schema.GroupVersionKind) apiextv1.CustomResourceDefinition {
	trueVal := true
	singular := strings.ToLower(gvk.Kind)
	plural := fmt.Sprintf("%ss", singular)
	return apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", plural, gvk.Group),
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: gvk.Group,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind:     gvk.Kind,
				ListKind: fmt.Sprintf("%sList", gvk.Kind),
				Singular: singular,
				Plural:   plural,
			},
			Scope: apiextv1.NamespaceScoped,
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name: "v1",
					Schema: &apiextv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextv1.JSONSchemaProps{
							Type:                   "object",
							XPreserveUnknownFields: &trueVal,
						},
					},
					Subresources: &apiextv1.CustomResourceSubresources{
						Status: &apiextv1.CustomResourceSubresourceStatus{},
					},
					Served:  true,
					Storage: true,
				},
			},
		},
	}
}

// BuildTestCR builds test CR
func BuildTestCR(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{"replicas": 2},
	}}
	obj.SetName("test")
	obj.SetNamespace("default")
	obj.SetGroupVersionKind(gvk)
	obj.SetUID("test-uid")
	obj.SetAnnotations(map[string]string{
		"helm.sdk.operatorframework.io/install-description":   "test install description",
		"helm.sdk.operatorframework.io/upgrade-description":   "test upgrade description",
		"helm.sdk.operatorframework.io/uninstall-description": "test uninstall description",
	})
	return obj
}
