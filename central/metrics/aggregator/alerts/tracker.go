package alerts

import (
	"context"
	"iter"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	alertDS "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
)

var getters = []common.LabelGetter[*storage.ListAlert]{
	{Label: "Cluster", Getter: func(alert *storage.ListAlert) string { return alert.GetCommonEntityInfo().GetClusterName() }},
	{Label: "Namespace", Getter: func(alert *storage.ListAlert) string { return alert.GetCommonEntityInfo().GetNamespace() }},
	{Label: "Resource", Getter: func(alert *storage.ListAlert) string { return alert.GetResource().GetName() }},
	{Label: "Deployment", Getter: func(alert *storage.ListAlert) string { return alert.GetDeployment().GetName() }},
	{Label: "IsDeploymentActive", Getter: isDeploymentActive},
	{Label: "Policy", Getter: func(alert *storage.ListAlert) string { return alert.GetPolicy().GetName() }},
	{Label: "Categories", Getter: func(alert *storage.ListAlert) string { return strings.Join(alert.GetPolicy().GetCategories(), ",") }},
	{Label: "Severity", Getter: func(alert *storage.ListAlert) string { return alert.GetPolicy().GetSeverity().String() }},
	{Label: "Action", Getter: func(alert *storage.ListAlert) string { return alert.GetEnforcementAction().String() }},
	{Label: "Stage", Getter: func(alert *storage.ListAlert) string { return alert.GetLifecycleStage().String() }},
	{Label: "EnforcementCount", Getter: func(alert *storage.ListAlert) string { return strconv.Itoa(int(alert.GetEnforcementCount())) }},
	{Label: "State", Getter: func(alert *storage.ListAlert) string { return alert.GetState().String() }},
}

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int)) *common.TrackerConfig[*storage.ListAlert] {
	return common.MakeTrackerConfig(
		"alerts",
		"aggregated policy violation alerts",
		getters,
		common.Bind3rd(trackAlertsMetrics, alertDS.Singleton()),
		gauge)
}

func isDeploymentActive(alert *storage.ListAlert) string {
	if alert.GetDeployment().GetInactive() {
		return "false"
	}
	return "true"
}

func trackAlertsMetrics(ctx context.Context, _ common.MetricLabelsExpressions, ds alertDS.DataStore) iter.Seq[*storage.ListAlert] {
	return func(yield func(*storage.ListAlert) bool) {
		// Optimization opportunity:
		// The resource filter is known at this point, so a more precise query
		// could be constructed here.
		_ = ds.WalkAll(ctx, func(alert *storage.ListAlert) error {
			if !yield(alert) {
				return common.ErrStopIterator
			}
			return nil
		})
	}
}
