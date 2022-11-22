package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/rox/image"
	auth "github.com/stackrox/rox/operator/apis/auth/v1alpha1"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/extensions"
	"github.com/stackrox/rox/operator/pkg/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	opts := []pkgReconciler.Option{
		pkgReconciler.WithExtraWatch(
			&source.Kind{Type: &platform.SecuredCluster{}},
			reconciler.HandleSiblings(platform.CentralGVK, mgr),
			// Only appearance and disappearance of a SecuredCluster resource can influence whether
			// an init bundle should be created by the Central controller.
			utils.CreateAndDeleteOnlyPredicate{},
		),
		// TODO(dhaus): need to check whether this is actually what we want to do, or it's something else.
		// Potentially, we could also do something similar such as creating a config map for _each_ CRD, but that does defy
		// the purpose of it I suppose.

		// I don't want central to be the one that will have the reconciliation loop in there, but keep the operator
		// for that.

		// Meaning, we have to somehow transfer the CRD data to a readable JSON representation of our proto.
		// It _could_ be done internally and sent via GRPC (authZ???), but initial proposal would be to de-couple those.

		// Obviously, this does create some bottle necks:
		// - If we have too many CRDs to watch, the operator potentially needs to scale.
		// - For storing the JSON representation within central, using ConfigMaps probably ain't it - why use CRD in the first place.
		// - Weird thing of having both things separate - CRs on one hand and then the "transformed" JSON representation.

		// Alternative would be the following:
		// - Have central implement a controller for the custom resource.
		// - Have a separate component (neither operator nor central itself) act as a controller for the CR - it might
		// 	 need us to be in the same pod context as central / a separate deployment with access to the postgres database.

		// Generally, I have to say, this does feel suboptimal.
		// Either way, we are introducing additional complexity to an already complex service (central), or we are adding
		// additional complexity by introducing another component, or we are having this weird config map problem with
		// the operator being the one doing some weird stuff there).

		pkgReconciler.WithExtraWatch(
			&source.Kind{Type: &auth.AuthProvider{}},
			reconciler.HandleSiblings(auth.AuthProviderGVK, mgr),
			utils.CreateAndDeleteOnlyPredicate{},
		),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralDBPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), extensions.PVCTargetCentral, extensions.DefaultCentralPVCName)),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), extensions.PVCTargetCentralDB, extensions.DefaultCentralDBPVCName)),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		pkgReconciler.WithReconcilePeriod(extensions.InitBundleReconcilePeriod),
		pkgReconciler.WithPauseReconcileAnnotation("stackrox.io/pause-reconcile"),
	}
	return reconciler.SetupReconcilerWithManager(
		mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.Translator{}, proxyEnv),
		opts...,
	)
}
