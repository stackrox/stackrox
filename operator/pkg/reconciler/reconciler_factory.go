package reconciler

import (
	"strconv"
	"strings"

	"github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/joelanford/helm-operator/pkg/values"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"github.com/stackrox/rox/pkg/env"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// defaultMaxReleaseHistorySize is the default maximum size of the Helm release history. The value of 10 is chosen
	// as a value that should be large enough in practice to allow meaningful manual investigations/recoveries, but
	// small enough such that overall space consumption will not be a concern.
	defaultMaxReleaseHistorySize = 10
)

var (
	maxReleaseHistorySizeSetting = env.RegisterSetting("ROX_MAX_HELM_RELEASE_HISTORY")
)

// SetupReconcilerWithManager creates and registers a new helm reconciler to the given controller manager.
func SetupReconcilerWithManager(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix string, translator values.Translator, extraOpts ...reconciler.Option) error {
	chart, err := image.GetDefaultImage().LoadChart(chartPrefix, charts.RHACSMetaValues())
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

	reconcilerOpts := []reconciler.Option{
		reconciler.WithChart(*chart),
		reconciler.WithGroupVersionKind(gvk),
		reconciler.WithValueTranslator(translator),
		//TODO(ROX-7362): re-evaluate enabling depended watches
		reconciler.SkipDependentWatches(true),
		reconciler.WithMaxReleaseHistory(maxReleaseHistorySize),
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
