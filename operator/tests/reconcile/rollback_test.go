package reconcile

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	centralTranslation "github.com/stackrox/rox/operator/internal/central/values/translation"
	stackroxReconciler "github.com/stackrox/rox/operator/internal/reconciler"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("Rolling back after failed upgrade", func() {
	const (
		centralName = "test-central"
		namespace   = "default"
	)

	var (
		mgr    manager.Manager
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		s := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(s)).To(Succeed())
		Expect(platform.AddToScheme(s)).To(Succeed())

		var err error
		mgr, err = manager.New(cfg, manager.Options{
			Scheme:  s,
			Metrics: server.Options{BindAddress: "0"},
		})
		Expect(err).NotTo(HaveOccurred())

		translator := centralTranslation.New(mgr.GetClient())
		Expect(stackroxReconciler.SetupReconcilerWithManager(
			mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
			translator, nil,
		)).To(Succeed())

		ctx, cancel = context.WithCancel(context.Background())
		go func() {
			defer GinkgoRecover()
			Expect(mgr.Start(ctx)).To(Succeed())
		}()
	})

	AfterEach(func() {
		By("cleaning up Central CR")
		central := &platform.Central{}
		objKey := types.NamespacedName{Namespace: namespace, Name: centralName}
		if err := mgr.GetAPIReader().Get(ctx, objKey, central); err == nil {
			central.SetFinalizers([]string{})
			_ = mgr.GetClient().Update(ctx, central)
			err = mgr.GetClient().Delete(ctx, central)
			if err != nil && !apiErrors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		}

		By("cleaning up PVCs")
		pvcList := &corev1.PersistentVolumeClaimList{}
		if err := mgr.GetAPIReader().List(ctx, pvcList, ctrlClient.InNamespace(namespace)); err == nil {
			for i := range pvcList.Items {
				_ = mgr.GetClient().Delete(ctx, &pvcList.Items[i])
			}
		}

		By("cleaning up PVs")
		pvList := &corev1.PersistentVolumeList{}
		if err := mgr.GetAPIReader().List(ctx, pvList); err == nil {
			for i := range pvList.Items {
				_ = mgr.GetClient().Delete(ctx, &pvList.Items[i])
			}
		}

		cancel()
	})

	It("should roll back successfully after a failed upgrade when bound PVCs exist", func() {
		By("creating a Central CR with Scanner V4 enabled")
		central := &platform.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:      centralName,
				Namespace: namespace,
			},
			Spec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: new(platform.ScannerV4ComponentEnabled),
				},
			},
		}
		Expect(mgr.GetClient().Create(ctx, central)).To(Succeed())

		By("waiting for initial install to complete")
		Eventually(func(g Gomega) {
			c := &platform.Central{}
			g.Expect(mgr.GetAPIReader().Get(ctx, types.NamespacedName{
				Namespace: namespace, Name: centralName,
			}, c)).To(Succeed())

			releaseFailed := c.GetCondition(platform.ConditionReleaseFailed)
			irreconcilable := c.GetCondition(platform.ConditionIrreconcilable)

			g.Expect(releaseFailed).NotTo(BeNil(), "ReleaseFailed condition should be set")
			g.Expect(string(releaseFailed.Status)).To(Equal(string(platform.StatusFalse)),
				"ReleaseFailed should be False after successful install")
			g.Expect(irreconcilable).NotTo(BeNil(), "Irreconcilable condition should be set")
			g.Expect(string(irreconcilable.Status)).To(Equal(string(platform.StatusFalse)),
				"Irreconcilable should be False after successful install")
		}, 2*time.Minute, 2*time.Second).Should(Succeed())

		By("simulating PVC binding by setting spec.volumeName")
		pvc := &corev1.PersistentVolumeClaim{}
		Expect(mgr.GetAPIReader().Get(ctx, types.NamespacedName{
			Namespace: namespace, Name: "scanner-v4-db",
		}, pvc)).To(Succeed())
		pvc.Spec.VolumeName = "fake-pv"
		Expect(mgr.GetClient().Update(ctx, pvc)).To(Succeed())

		By("applying an invalid overlay to trigger upgrade failure")
		Expect(mgr.GetAPIReader().Get(ctx, types.NamespacedName{
			Namespace: namespace, Name: centralName,
		}, central)).To(Succeed())
		central.Spec.Overlays = []*platform.K8sObjectOverlay{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "scanner-v4-matcher-config",
				Patches: []*platform.K8sObjectOverlayPatch{
					{
						Path:  `data.config\.yaml`,
						Value: "matcher:\n  vulnerabilities_url: \"https://bad\"", // This will be unmarshalled as a map.
					},
				},
			},
		}
		Expect(mgr.GetClient().Update(ctx, central)).To(Succeed())

		By("waiting for upgrade failure with successful rollback")
		Eventually(func(g Gomega) {
			c := &platform.Central{}
			g.Expect(mgr.GetAPIReader().Get(ctx, types.NamespacedName{
				Namespace: namespace, Name: centralName,
			}, c)).To(Succeed())

			releaseFailed := c.GetCondition(platform.ConditionReleaseFailed)
			g.Expect(releaseFailed).NotTo(BeNil(), "ReleaseFailed condition should exist")
			g.Expect(string(releaseFailed.Status)).To(Equal(string(platform.StatusTrue)),
				"ReleaseFailed should be True after failed upgrade")
			g.Expect(releaseFailed.Message).NotTo(ContainSubstring("rollback failed"),
				"Rollback should succeed with Force=false; got: %s", releaseFailed.Message)

			// Reduce log noise.
			messageTruncated := releaseFailed.Message[:min(len(releaseFailed.Message), 100)] + " [...]"
			GinkgoWriter.Printf("ReleaseFailed condition: status=%s, reason=%s, message='%s'\n",
				releaseFailed.Status, releaseFailed.Reason, messageTruncated)
		}, 2*time.Minute, 2*time.Second).Should(Succeed())

		By("verifying PVC is still intact")
		Expect(mgr.GetAPIReader().Get(ctx, types.NamespacedName{
			Namespace: namespace, Name: "scanner-v4-db",
		}, pvc)).To(Succeed())
		Expect(pvc.Spec.VolumeName).To(Equal("fake-pv"))
	})
})
