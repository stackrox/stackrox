package policy_violations

import (
	"context"
	"iter"
	"strconv"
	"strings"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
)

var labelOrder = common.MakeLabelOrderMap([]common.Label{
	"Cluster",
	"Namespace",
	"Deployment",
	"IsActiveDeployment",
	"Resource",
	"Policy",
	"Categories",
	"Severity",
	"Action",
	"Stage",
	"EnforcementCount",
	"State",
})

func MakeTrackerConfig() *common.TrackerConfig {
	return common.MakeTrackerConfig("violations", "aggregated policy violations",
		labelOrder, common.Bind3rd(trackViolationsMetrics, alertDS.Singleton()))
}

func trackViolationsMetrics(ctx context.Context, _ common.MetricLabelsExpressions, ds alertDS.DataStore) iter.Seq[common.Finding] {
	return func(yield func(common.Finding) bool) {
		// Optimization opportunity:
		// The resource filter is known at this point, so a more precise query could be constructed here.
		_ = ds.WalkAll(ctx, func(alert *storage.ListAlert) error {
			if !yield(makeFinding(alert)) {
				return common.ErrStopIterator
			}
			return nil
		})
	}
}

var getters = map[common.Label]func(alert *storage.ListAlert) string{
	"Cluster":    func(alert *storage.ListAlert) string { return alert.GetCommonEntityInfo().GetClusterName() },
	"Namespace":  func(alert *storage.ListAlert) string { return alert.GetCommonEntityInfo().GetNamespace() },
	"Resource":   func(alert *storage.ListAlert) string { return alert.GetResource().GetName() },
	"Deployment": func(alert *storage.ListAlert) string { return alert.GetDeployment().GetName() },
	"IsActiveDeployment": func(alert *storage.ListAlert) string {
		if alert.GetDeployment().GetInactive() {
			return "false"
		}
		return "true"
	},
	"Policy":           func(alert *storage.ListAlert) string { return alert.GetPolicy().GetName() },
	"Categories":       func(alert *storage.ListAlert) string { return strings.Join(alert.GetPolicy().GetCategories(), ",") },
	"Severity":         func(alert *storage.ListAlert) string { return alert.GetPolicy().GetSeverity().String() },
	"Action":           func(alert *storage.ListAlert) string { return alert.GetEnforcementAction().String() },
	"Stage":            func(alert *storage.ListAlert) string { return alert.GetLifecycleStage().String() },
	"EnforcementCount": func(alert *storage.ListAlert) string { return strconv.Itoa(int(alert.GetEnforcementCount())) },
	"State":            func(alert *storage.ListAlert) string { return alert.GetState().String() },
}

func makeFinding(alert *storage.ListAlert) common.Finding {
	return func(label common.Label) string {
		return getters[label](alert)
	}
}
