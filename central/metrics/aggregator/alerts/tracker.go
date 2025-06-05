package alerts

import (
	"context"
	"iter"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	alertDS "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var getters = []common.LabelGetter[*finding]{
	// Alert
	{Label: "Cluster", Getter: func(f *finding) string { return f.GetClusterName() }},
	{Label: "Namespace", Getter: func(f *finding) string { return f.GetNamespace() }},
	{Label: "Resource", Getter: func(f *finding) string { return f.GetResource().GetName() }},
	{Label: "Deployment", Getter: func(f *finding) string { return f.GetDeployment().GetName() }},
	{Label: "IsDeploymentActive", Getter: func(f *finding) string { return strconv.FormatBool(!f.GetDeployment().GetInactive()) }},
	{Label: "IsPlatformComponent", Getter: func(f *finding) string { return strconv.FormatBool(f.GetPlatformComponent()) }},
	{Label: "Policy", Getter: func(f *finding) string { return f.GetPolicy().GetName() }},
	{Label: "Categories", Getter: func(f *finding) string { return strings.Join(f.GetPolicy().GetCategories(), ",") }},
	{Label: "Severity", Getter: func(f *finding) string { return f.GetPolicy().GetSeverity().String() }},
	{Label: "Action", Getter: func(f *finding) string { return f.GetEnforcement().GetAction().String() }},
	{Label: "Message", Getter: func(f *finding) string { return f.GetEnforcement().GetMessage() }},
	{Label: "Stage", Getter: func(f *finding) string { return f.GetLifecycleStage().String() }},
	{Label: "State", Getter: func(f *finding) string { return f.GetState().String() }},
	{Label: "Entity", Getter: func(f *finding) string { return f.GetEntityType().String() }},
	{Label: "EntityName", Getter: getEntityName},

	// Violation
	{Label: "Type", Getter: func(f *finding) string { return f.GetType().String() }},
}

type finding struct {
	common.OneOrMore
	*storage.Alert
	*storage.Alert_Violation
}

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int)) *common.TrackerConfig[*finding] {
	return common.MakeTrackerConfig(
		"alerts",
		"aggregated policy violation alerts",
		getters,
		common.Bind4th(trackAlertsMetrics, alertDS.Singleton()),
		gauge)
}

func trackAlertsMetrics(ctx context.Context, query *v1.Query, _ common.MetricsConfiguration, ds alertDS.DataStore) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		_ = ds.WalkByQuery(ctx, query, func(a *storage.Alert) error {
			f.Alert = a
			for _, v := range a.GetViolations() {
				f.Alert_Violation = v
				if !yield(&f) {
					return common.ErrStopIterator
				}
			}
			return nil
		})
	}
}

func getEntityName(f *finding) string {
	switch e := f.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return e.Deployment.GetName()
	case *storage.Alert_Image:
		return e.Image.GetName().GetFullName()
	case *storage.Alert_Resource_:
		return e.Resource.GetName()
	}
	return ""
}
