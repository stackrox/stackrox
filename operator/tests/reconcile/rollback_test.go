package reconcile

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmClient "github.com/operator-framework/helm-operator-plugins/pkg/client"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	centralTranslation "github.com/stackrox/rox/operator/internal/central/values/translation"
	stackroxReconciler "github.com/stackrox/rox/operator/internal/reconciler"
	"github.com/stackrox/rox/pkg/uuid"
)

var _ = Describe("Upgrade failure with rollbacks disabled", func() {
	const (
		centralName = "test-central"
	)

	var (
		namespace             = "upgrade-failure-test-" + uuid.NewV4().String()[0:6]
		namespacedCentralName = types.NamespacedName{Namespace: namespace, Name: centralName}

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

		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		Expect(mgr.GetClient().Create(ctx, &ns)).To(Succeed())
	})

	AfterEach(func() {
		By("ensuring Central CR and test namespace is deleted")
		ns := corev1.Namespace{}
		if err := mgr.GetAPIReader().Get(ctx, types.NamespacedName{Name: namespace}, &ns); err == nil {
			central := &platform.Central{}
			if err := mgr.GetAPIReader().Get(ctx, namespacedCentralName, central); err == nil {
				By("deleting finalizers")
				central.SetFinalizers([]string{})
				Expect(mgr.GetClient().Update(ctx, central)).To(Succeed())
				By("marking CR for deletion")
				Expect(mgr.GetClient().Delete(ctx, central)).To(Succeed())
				By("ensuring CR is gone")
				central := platform.Central{}
				Eventually(func(g Gomega) {
					g.Expect(mgr.GetAPIReader().Get(ctx, namespacedCentralName, &central)).
						To(MatchError(apiErrors.IsNotFound, "IsNotFound"))
				}, 10*time.Second, 1*time.Second).To(Succeed())
			}
			By("cleaning up test namespace")
			Expect(mgr.GetClient().Delete(ctx, &ns)).To(Succeed())
		}

		cancel() // Stops manager.
	})

	It("should not attempt rollback after a failed upgrade, preserving PVCs", func() {
		pvcName := types.NamespacedName{Namespace: namespace, Name: "scanner-v4-db"}
		var pvcUid types.UID
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

		By("creating a Central CR with Scanner V4 enabled")
		createCentral(mgr, ctx, central)

		By("waiting for initial install to complete")
		waitForInstall(mgr, ctx, namespacedCentralName)

		By("simulating PVC binding by setting spec.volumeName")
		simulatePVCBinding(mgr, ctx, pvcName, &pvcUid)

		By("applying an invalid overlay to trigger upgrade failure")
		applyInvalidOverlay(mgr, ctx, namespacedCentralName)

		By("waiting for upgrade failure without rollback attempt")
		waitForUpgradeFailure(mgr, ctx, namespacedCentralName)

		By("verifying no rollback revision in Helm release history")
		verifyNoRollbackRevision(mgr, ctx, central, centralName)

		By("repairing the custom resource, which failed reconciliation")
		repairCR(mgr, ctx, central, namespacedCentralName)

		By("waiting for reconciliation to succeed again")
		waitForRecovery(mgr, ctx, namespacedCentralName)

		By("verifying PVC is still intact")
		verifyPVCIntact(mgr, ctx, pvcName, pvcUid)
	})
})

func createCentral(mgr manager.Manager, ctx context.Context, central *platform.Central) {
	Expect(mgr.GetClient().Create(ctx, central)).To(Succeed())
}

func waitForInstall(mgr manager.Manager, ctx context.Context, key types.NamespacedName) {
	Eventually(func(g Gomega) {
		c := &platform.Central{}
		g.Expect(mgr.GetAPIReader().Get(ctx, key, c)).To(Succeed())
		g.Expect(c.GetCondition(platform.ConditionDeployed)).To(
			SatisfyAll(Not(BeNil()), HaveField("Status", Equal(platform.StatusTrue))),
			"Deployed condition should be True after successful install")
	}, 30*time.Second, 2*time.Second).Should(Succeed())
}

func simulatePVCBinding(mgr manager.Manager, ctx context.Context, pvcName types.NamespacedName, pvcUid *types.UID) {
	Eventually(func(g Gomega) {
		pvc := &corev1.PersistentVolumeClaim{}
		g.Expect(mgr.GetAPIReader().Get(ctx, pvcName, pvc)).To(Succeed())
		pvc.Spec.VolumeName = "fake-pv"
		g.Expect(mgr.GetClient().Update(ctx, pvc)).To(Succeed())
		*pvcUid = pvc.GetUID()
	}, 10*time.Second, 1*time.Second).Should(Succeed())
}

func applyInvalidOverlay(mgr manager.Manager, ctx context.Context, key types.NamespacedName) {
	Eventually(func(g Gomega) {
		c := &platform.Central{}
		g.Expect(mgr.GetAPIReader().Get(ctx, key, c)).To(Succeed())
		c.Spec.Overlays = []*platform.K8sObjectOverlay{
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
		g.Expect(mgr.GetClient().Update(ctx, c)).To(Succeed())
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}

func waitForUpgradeFailure(mgr manager.Manager, ctx context.Context, key types.NamespacedName) {
	Eventually(func(g Gomega) {
		c := &platform.Central{}
		g.Expect(mgr.GetAPIReader().Get(ctx, key, c)).To(Succeed())

		releaseFailed := c.GetCondition(platform.ConditionReleaseFailed)
		irreconcilable := c.GetCondition(platform.ConditionIrreconcilable)

		g.Expect(releaseFailed).To(SatisfyAll(Not(BeNil()), HaveField("Status", Equal(platform.StatusTrue))),
			"ReleaseFailed condition should be True")
		g.Expect(irreconcilable).To(SatisfyAll(Not(BeNil()), HaveField("Status", Equal(platform.StatusTrue))),
			"Irreconcilable condition should be True")
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}

func verifyNoRollbackRevision(mgr manager.Manager, ctx context.Context, central *platform.Central, releaseName string) {
	actionConfigGetter, err := helmClient.NewActionConfigGetter(mgr.GetConfig(), mgr.GetRESTMapper())
	Expect(err).NotTo(HaveOccurred())
	acfg, err := actionConfigGetter.ActionConfigFor(ctx, central)
	Expect(err).NotTo(HaveOccurred())
	history, err := acfg.Releases.History(releaseName)
	Expect(err).NotTo(HaveOccurred())
	for _, rel := range history {
		Expect(rel.Info.Status).NotTo(Equal(release.StatusPendingRollback),
			"release v%d should not have pending-rollback status", rel.Version)
		Expect(rel.Info.Description).NotTo(MatchRegexp(`(?i)rollback`),
			"release v%d description %q suggests a rollback was attempted", rel.Version, rel.Info.Description)
	}
}

func repairCR(mgr manager.Manager, ctx context.Context, central *platform.Central, key types.NamespacedName) {
	Eventually(func(g Gomega) {
		g.Expect(mgr.GetAPIReader().Get(ctx, key, central)).To(Succeed())
		central.Spec.Overlays = []*platform.K8sObjectOverlay{}
		g.Expect(mgr.GetClient().Update(ctx, central)).To(Succeed())
	}, 10*time.Second, 1*time.Second).Should(Succeed())
}

func waitForRecovery(mgr manager.Manager, ctx context.Context, key types.NamespacedName) {
	Eventually(func(g Gomega) {
		c := &platform.Central{}
		g.Expect(mgr.GetAPIReader().Get(ctx, key, c)).To(Succeed())

		releaseFailed := c.GetCondition(platform.ConditionReleaseFailed)
		irreconcilable := c.GetCondition(platform.ConditionIrreconcilable)

		g.Expect(c.GetCondition(platform.ConditionDeployed)).To(
			SatisfyAll(Not(BeNil()), HaveField("Status", Equal(platform.StatusTrue))),
			"Deployed condition should be True again")
		g.Expect(releaseFailed).To(SatisfyAny(BeNil(), HaveField("Status", Equal(platform.StatusFalse))),
			"ReleaseFailed condition should be nil or False")
		g.Expect(irreconcilable).To(SatisfyAny(BeNil(), HaveField("Status", Equal(platform.StatusFalse))),
			"Irreconcilable condition should be nil or False")
	}, 2*time.Minute, 2*time.Second).Should(Succeed())
}

func verifyPVCIntact(mgr manager.Manager, ctx context.Context, pvcName types.NamespacedName, pvcUid types.UID) {
	pvc := &corev1.PersistentVolumeClaim{}
	Expect(mgr.GetAPIReader().Get(ctx, pvcName, pvc)).To(Succeed())
	Expect(pvc.Spec.VolumeName).To(Equal("fake-pv"))
	Expect(pvc.GetUID()).To(Equal(pvcUid))
}
