package reconciler

import (
	"strconv"
	"strings"
	"time"

	"github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/joelanford/helm-operator/pkg/values"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
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
	if !buildinfo.ReleaseBuild {
		metaVals.MainRegistry = mainRegistryOverride.Setting()
		metaVals.CollectorRegistry = collectorRegistryOverride.Setting()
	}
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

	reconcilerOpts := []reconciler.Option{
		reconciler.WithChart(*chart),
		reconciler.WithGroupVersionKind(gvk),
		reconciler.WithValueTranslator(translator),
		// TODO(ROX-7362): re-evaluate enabling depended watches
		reconciler.SkipDependentWatches(true),
		reconciler.WithMaxReleaseHistory(maxReleaseHistorySize),
		reconciler.WithMarkFailedAfter(markReleaseFailedAfter),
	}
	reconcilerOpts = append(reconcilerOpts, extraOpts...)

	r, err := reconciler.New(reconcilerOpts...)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s reconciler", gvk)
	}

	if err := r.SetupWithManager(mgr, reconciler.SetupOpts{DisableSetupScheme: true}); err != nil {
		return errors.Wrapf(err, "unable to setup %s reconciler", gvk)
	}
	return nil
}
