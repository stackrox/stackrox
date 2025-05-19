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
		labelOrder, common.Bind2nd(trackViolationsMetrics, alertDS.Singleton()))
}

func trackViolationsMetrics(ctx context.Context, ds alertDS.DataStore) iter.Seq[common.Finding] {
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

func makeFinding(alert *storage.ListAlert) common.Finding {
	alert.GetPolicy().GetCategories()
	return func(label common.Label) string {
		switch label {
		case "Cluster":
			return alert.GetCommonEntityInfo().GetClusterName()
		case "Namespace":
			return alert.GetCommonEntityInfo().GetNamespace()
		case "Resource":
			return alert.GetResource().GetName()
		case "Deployment":
			return alert.GetDeployment().GetName()
		case "IsActiveDeployment":
			if alert.GetDeployment().GetInactive() {
				return "false"
			}
			return "true"
		case "Policy":
			return alert.GetPolicy().GetName()
		case "Categories":
			return strings.Join(alert.GetPolicy().GetCategories(), ",")
		case "Severity":
			return alert.GetPolicy().GetSeverity().String()
		case "Action":
			return alert.GetEnforcementAction().String()
		case "Stage":
			return alert.GetLifecycleStage().String()
		case "EnforcementCount":
			return strconv.Itoa(int(alert.GetEnforcementCount()))
		case "State":
			return alert.GetState().String()
		default:
			return ""
		}
	}
}
