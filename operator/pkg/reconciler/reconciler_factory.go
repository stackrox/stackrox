package reconciler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/operator-framework/helm-operator-plugins/pkg/client"
	"github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/operator/pkg/overlays"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/postrender"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// defaultMaxReleaseHistorySize is the default maximum size of the Helm release history. The value of 10 is chosen
	// as a value that should be large enough in practice to allow meaningful manual investigations/recoveries, but
	// small enough such that overall space consumption will not be a concern.
	defaultMaxReleaseHistorySize = 10

	// defaultMarkReleaseFailedAfter is the default time after which a release that is seemingly stuck in a
	// pending/locked state is marked failed.
	defaultMarkReleaseFailedAfter = 2 * time.Minute
)

var (
	maxReleaseHistorySizeSetting  = env.RegisterSetting("ROX_MAX_HELM_RELEASE_HISTORY")
	markReleaseFailedAfterSetting = env.RegisterSetting("ROX_MARK_RELEASE_FAILED_AFTER")
)

// SetupReconcilerWithManager creates and registers a new helm reconciler to the given controller manager.
func SetupReconcilerWithManager(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix image.ChartPrefix, translator values.Translator, extraOpts ...reconciler.Option) error {
	metaVals := charts.GetMetaValuesForFlavor(defaults.GetImageFlavorFromEnv())
	metaVals.Operator = true

	metaVals.ImagePullSecrets.AllowNone = true

	chart, err := image.GetDefaultImage().LoadChart(chartPrefix, metaVals)
	if err != nil {
		return err
	}

	maxReleaseHistorySize := defaultMaxReleaseHistorySize
	if maxHistoryStr := strings.TrimSpace(maxReleaseHistorySizeSetting.Setting()); maxHistoryStr != "" {
		maxReleaseHistorySize, err = strconv.Atoi(maxHistoryStr)
		if err != nil {
			return errors.Wrapf(err, "invalid %s setting", maxReleaseHistorySizeSetting.EnvVar())
		}
	}

	markReleaseFailedAfter := defaultMarkReleaseFailedAfter
	if markReleaseFailedAfterStr := strings.TrimSpace(markReleaseFailedAfterSetting.Setting()); markReleaseFailedAfterStr != "" {
		markReleaseFailedAfter, err = time.ParseDuration(markReleaseFailedAfterStr)
		if err != nil {
			return errors.Wrapf(err, "invalid %s setting", markReleaseFailedAfterSetting.EnvVar())
		}
	}

	logger := ctrl.Log.WithName("controllers").WithName(gvk.Kind)

	actionConfigGetter, err := client.NewActionConfigGetter(mgr.GetConfig(), mgr.GetRESTMapper(), logger)
	if err != nil {
		return fmt.Errorf("creating action config getter: %w", err)
	}

	var opts = []client.ActionClientGetterOption{
		client.AppendPostRenderers(func(rm meta.RESTMapper, kubeClient kube.Interface, obj ctrlClient.Object) postrender.PostRenderer {
			return overlays.NewOverlayPostRenderer(obj, obj.GetNamespace())
		}),
	}

	actionClientGetter, err := client.NewActionClientGetter(actionConfigGetter, opts...)
	if err != nil {
		return errors.Wrap(err, "unable to create action client getter")
	}

	reconcilerOpts := []reconciler.Option{
		reconciler.WithChart(*chart),
		reconciler.WithGroupVersionKind(gvk),
		reconciler.WithValueTranslator(translator),
		// TODO(ROX-7362): re-evaluate enabling depended watches
		reconciler.SkipDependentWatches(true),
		reconciler.WithMaxReleaseHistory(maxReleaseHistorySize),
		reconciler.WithMarkFailedAfter(markReleaseFailedAfter),
		reconciler.SkipPrimaryGVKSchemeRegistration(true),
		reconciler.WithLog(logger),
		reconciler.WithActionClientGetter(actionClientGetter),
	}
	reconcilerOpts = append(reconcilerOpts, extraOpts...)

	r, err := reconciler.New(reconcilerOpts...)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s reconciler", gvk)
	}

	if err := r.SetupWithManager(mgr); err != nil {
		return errors.Wrapf(err, "unable to setup %s reconciler", gvk)
	}
	return nil
}

// HandleSiblings returns an event handler which generates reconcile requests for
// every (in our case typically one) resource of specified type, which resides in the same namespace as the
// observed resource.
func HandleSiblings(gvk schema.GroupVersionKind, manager ctrl.Manager) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object ctrlClient.Object) []reconcile.Request {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		utils.ListSiblings(ctx, list, object, manager.GetClient())
		var ret []reconcile.Request
		for _, c := range list.Items {
			ret = append(ret, utils.RequestFor(&c)) // #nosec
		}
		return ret
	})
}
