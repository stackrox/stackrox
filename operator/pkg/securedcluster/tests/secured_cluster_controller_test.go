package tests

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Secured Cluster Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1

	Context("Plain installation", func() {
		It("Should install successfully", func() {
			created := &v1alpha1.SecuredCluster{
				Spec: v1alpha1.SecuredClusterSpec{
					ClusterName: "testing-cluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "stackrox",
					Name:      "stackrox-secured-cluster-services",
				},
			}

			Expect(k8sClient.Create(context.Background(), created)).Should(Succeed())

			By("Should create a sensor deployment")
			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: "sensor", Namespace: "stackrox"}, &v1.Deployment{})
			}, timeout, interval).ShouldNot(HaveOccurred())

			By("Deleting secured cluster resource")
			Eventually(func() error {
				return k8sClient.Delete(context.Background(), &v1alpha1.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox", Name: "stackrox-secured-cluster-services"}},
				)
			}, timeout, interval).ShouldNot(HaveOccurred())

			//TODO: Validate cluster was created in central
			//TODO: Validate cluster can communicate with central
			//TODO: Validate Sensor, Admission and Collector are healthy
			//TODO: Wait for central cluster health to be healthy

			By("CustomResource was deleted in k8s")
			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: "stackrox-secured-cluster-services", Namespace: "stackrox"}, &v1alpha1.SecuredCluster{})
			}, timeout, interval).ShouldNot(BeNil())

			By("Sensor was deleted in k8s")
			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: "sensor", Namespace: "stackrox"}, &v1.Deployment{})
			}, timeout, interval).ShouldNot(BeNil())
		})
	})
})
