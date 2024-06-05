package reconcile

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	mapkubeapisCommon "github.com/helm/helm-mapkubeapis/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmClient "github.com/operator-framework/helm-operator-plugins/pkg/client"
	"github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const mapFile = "../../pkg/config/mapkubeapis/Map.yaml"

var (
	pspTemplate = `apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: stackrox-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
    - 'hostPath'

  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAs'
    ranges:
      - min: 4000
        max: 4000
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'MustRunAs'
    ranges:
      - min: 4000
        max: 4000
`
	chartWithPSP = chart.Chart{
		Metadata: &chart.Metadata{
			Name:       "stackrox-test",
			Version:    "0.1.0-alpha.1",
			AppVersion: "1.0",
		},
		Templates: []*chart.File{
			{Name: "templates/psp.tpl", Data: []byte(pspTemplate)},
		},
	}
	chartWithoutPSP = chart.Chart{
		Metadata: &chart.Metadata{
			Name:       "stackrox-test",
			Version:    "0.1.0-beta.2",
			AppVersion: "1.1",
		},
		Templates: []*chart.File{},
	}
)

var _ = Describe("MapKubeAPIsExtension", func() {
	Describe("Execute", func() {
		When("Upgrading the cluster with an existing release", func() {
			var (
				obj    *unstructured.Unstructured
				objKey types.NamespacedName
				req    reconcile.Request

				mgr    manager.Manager
				ctx    context.Context
				cancel context.CancelFunc

				ac                 helmClient.ActionInterface
				actionClientGetter helmClient.ActionClientGetter
				store              *storage.Storage
			)

			BeforeEach(func() {
				mgr = getManagerOrFail()
				ctx, cancel = context.WithCancel(context.Background())
				go func() { Expect(mgr.GetCache().Start(ctx)) }()
				Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())

				obj = BuildTestCR(gvk)
				objKey = types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
				req = reconcile.Request{NamespacedName: objKey}
				Expect(mgr.GetClient().Create(ctx, obj)).To(Succeed())

				actionConfigGetter, err := helmClient.NewActionConfigGetter(mgr.GetConfig(), mgr.GetRESTMapper(), logr.Discard())
				Expect(err).NotTo(HaveOccurred())
				acfg, err := actionConfigGetter.ActionConfigFor(obj)
				Expect(err).NotTo(HaveOccurred())
				store = acfg.Releases
				actionClientGetter, err := helmClient.NewActionClientGetter(actionConfigGetter)
				Expect(err).NotTo(HaveOccurred())
				ac, err = actionClientGetter.ActionClientFor(obj)
				Expect(err).NotTo(HaveOccurred())
			})

			installWithPSP := func() {
				install := action.NewInstall(&action.Configuration{
					Releases: store,
				})
				install.ClientOnly = true
				install.ReleaseName = obj.GetName()
				install.Namespace = obj.GetNamespace()
				rel, err := install.Run(&chartWithPSP, map[string]interface{}{})
				Expect(err).NotTo(HaveOccurred())
				Expect(store.Create(rel)).To(Succeed())
			}

			When("PSP is not supported on the new cluster", func() {
				BeforeEach(func() {
					_, err := mgr.GetRESTMapper().ResourceFor(schema.GroupVersionResource{
						Group:    "policy",
						Version:  "v1beta1",
						Resource: "podsecuritypolicies",
					})
					Expect(err).To(HaveOccurred())
				})

				It("should recover after the release is installed", func() {
					installWithPSP()
					r := newReconciler(chartWithoutPSP, actionClientGetter, mgr)
					result, err := r.Reconcile(ctx, req)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
				})
			})

			AfterEach(func() {
				By("ensuring the release is uninstalled", func() {
					if _, err := ac.Get(obj.GetName()); errors.Is(err, driver.ErrReleaseNotFound) {
						return
					}
					_, err := ac.Uninstall(obj.GetName())
					if err != nil {
						panic(err)
					}
				})

				By("ensuring the CR is deleted", func() {
					err := mgr.GetAPIReader().Get(ctx, objKey, obj)
					if apiErrors.IsNotFound(err) {
						return
					}
					Expect(err).NotTo(HaveOccurred())
					obj.SetFinalizers([]string{})
					Expect(mgr.GetClient().Update(ctx, obj)).To(Succeed())
					err = mgr.GetClient().Delete(ctx, obj)
					if apiErrors.IsNotFound(err) {
						return
					}
					Expect(err).NotTo(HaveOccurred())
				})
				cancel()
			})
		})
	})

})

func newReconciler(chrt chart.Chart, actionClientGetter helmClient.ActionClientGetter, mgr manager.Manager) *reconciler.Reconciler {
	r, err := reconciler.New(
		reconciler.WithChart(chrt),
		reconciler.WithGroupVersionKind(gvk),
		reconciler.WithActionClientGetter(actionClientGetter),
		reconciler.WithPreExtension(commonExtensions.MapKubeAPIsExtension(commonExtensions.MapKubeAPIsExtensionConfig{
			KubeConfig: getKubeConfig(),
			MapFile:    mapFile,
		})),
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(r.SetupWithManager(mgr)).To(Succeed())
	return r
}

func getManagerOrFail() manager.Manager {
	mgr, err := manager.New(cfg, manager.Options{
		Metrics: server.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())
	return mgr
}

func getKubeConfig() mapkubeapisCommon.KubeConfig {
	user, err := testEnv.AddUser(envtest.User{Name: "test", Groups: []string{"system:masters"}}, &rest.Config{})
	Expect(err).NotTo(HaveOccurred())

	// cleaning this up is handled when our tmpDir is deleted
	out, err := os.CreateTemp(testEnv.ControlPlane.APIServer.CertDir, "*.kubecfg")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		_ = out.Close()
	}()
	contents, err := user.KubeConfig()
	Expect(err).NotTo(HaveOccurred())
	_, err = out.Write(contents)
	Expect(err).NotTo(HaveOccurred())

	return mapkubeapisCommon.KubeConfig{
		File: out.Name(),
	}
}
