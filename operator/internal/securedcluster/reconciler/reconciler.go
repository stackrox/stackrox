package reconciler

import (
	"context"
	"fmt"
	"time"

	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	statusController "github.com/stackrox/rox/operator/internal/common/status"
	"github.com/stackrox/rox/operator/internal/legacy"
	"github.com/stackrox/rox/operator/internal/proxy"
	"github.com/stackrox/rox/operator/internal/reconciler"
	"github.com/stackrox/rox/operator/internal/securedcluster/extensions"
	scTranslation "github.com/stackrox/rox/operator/internal/securedcluster/values/translation"
	"github.com/stackrox/rox/operator/internal/tlsprofile"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/operator/internal/values/translation"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/securedcluster"
	"github.com/stackrox/rox/pkg/version"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// A Central in any namespace changes the derived cluster label for every SecuredCluster.
func securedClusterRequests(ctx context.Context, client ctrlClient.Client) []reconcile.Request {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	list := &platform.SecuredClusterList{}
	if err := client.List(ctx, list); err != nil {
		// Match HandleSiblings: a failed list must not silently leave resources stale.
		panic(fmt.Errorf("cannot retrieve SecuredClusters when processing Central event: %w", err))
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		requests = append(requests, utils.RequestFor(&list.Items[i]))
	}
	return requests
}

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, selector string, tlsProfile *tlsprofile.TLSProfile) error {
	renderCache := rendercache.NewRenderCache()
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	extraEventWatcher := pkgReconciler.WithExtraWatch(
		source.Kind[*platform.Central](
			mgr.GetCache(),
			&platform.Central{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, _ *platform.Central) []reconcile.Request {
				return securedClusterRequests(ctx, mgr.GetClient())
			}),
			// Central creation and deletion affect the cluster-wide derived label; a
			// Central in the same namespace also affects local scanner detection.
			utils.CreateAndDeleteOnlyPredicate[*platform.Central]{}))
	// IMPORTANT: The FeatureDefaultingExtension preExtensions implements feature-defaulting logic
	// and therefore must be executed and registered first.
	// New extensions shall be added to otherPreExtensions to guarantee this ordering.
	otherPreExtensions := []pkgReconciler.Option{
		pkgReconciler.WithPreExtension(extensions.CheckClusterNameExtension()),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), mgr.GetAPIReader(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(extensions.ReconcileLocalScannerDBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileLocalScannerV4DBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.SensorCAHashExtension(mgr.GetClient(), mgr.GetAPIReader(), renderCache)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
	}

	// Plug in custom event predicate to skip reconciliation for updates caused by the status controller.
	// to prevent unnecessary Helm dry-runs.
	predicates := []pkgReconciler.Option{
		pkgReconciler.WithPredicate(statusController.NewSkipStatusControllerUpdates(mgr.GetLogger(), platform.SecuredClusterGVK.Kind)),
	}

	opts := make([]pkgReconciler.Option, 0, len(otherPreExtensions)+len(predicates)+8)
	opts = append(opts, extraEventWatcher)
	// Watch the CABundle ConfigMap that Sensor creates
	opts = append(opts, pkgReconciler.WithExtraWatch(
		source.Kind(
			mgr.GetCache(),
			&corev1.ConfigMap{},
			reconciler.HandleSiblings[*corev1.ConfigMap](platform.SecuredClusterGVK, mgr),
			&utils.ResourceWithNamePredicate[*corev1.ConfigMap]{
				Name: pkgKubernetes.TLSCABundleConfigMapName,
			},
		),
	))
	// Watch the Sensor TLS secret that triggers rollout restarts when CA rotation occurs.
	opts = append(opts, pkgReconciler.WithExtraWatch(
		source.Kind(
			mgr.GetCache(),
			&corev1.Secret{},
			reconciler.HandleSiblings[*corev1.Secret](platform.SecuredClusterGVK, mgr),
			&utils.ResourceWithNamePredicate[*corev1.Secret]{
				Name: securedcluster.SensorTLSSecretName,
			},
		),
	))
	opts = append(opts, pkgReconciler.WithPreExtension(extensions.VerifyCollisionFreeSecuredCluster(mgr.GetClient())))
	opts = append(opts, pkgReconciler.WithPreExtension(extensions.FeatureDefaultingExtension(mgr.GetClient())))
	opts = append(opts, otherPreExtensions...)
	opts = append(opts, predicates...)
	opts = append(opts, pkgReconciler.WithAggressiveConflictResolution(true))
	opts = append(opts, pkgReconciler.WithPauseReconcileAnnotation(commonExtensions.PauseReconcileAnnotation))
	opts, err := commonExtensions.AddSelectorOptionIfNeeded(selector, opts)
	if err != nil {
		return err
	}

	opts = commonExtensions.AddMapKubeAPIsExtensionIfMapFileExists(opts, mgr.GetRESTMapper())

	// Using uncached UncachedClient since this is reading secrets not
	// owned by the operator so we can't guarantee labels for cache
	// are set properly.
	pullSecretRefInjector := legacy.NewImagePullSecretReferenceInjector(
		mgr.GetAPIReader(), "imagePullSecrets",
		"secured-cluster-services-main", "stackrox", "stackrox-scanner", "stackrox-scanner-v4")
	pullSecretRefInjector = pullSecretRefInjector.WithExtraImagePullSecrets(
		"mainImagePullSecrets", "secured-cluster-services-main", "stackrox")
	pullSecretRefInjector = pullSecretRefInjector.WithExtraImagePullSecrets(
		"collectorImagePullSecrets", "secured-cluster-services-collector", "stackrox", "collector-stackrox")

	return reconciler.SetupReconcilerWithManager(
		mgr, platform.SecuredClusterGVK,
		image.SecuredClusterServicesChartPrefix,
		translation.WithEnrichment(
			scTranslation.New(mgr.GetClient(), mgr.GetAPIReader()),
			proxy.NewProxyEnvVarsInjector(proxyEnv, mgr.GetLogger()),
			tlsprofile.NewEnricher(tlsProfile),
			pullSecretRefInjector,
		),
		renderCache,
		opts...,
	)
}
