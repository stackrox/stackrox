package clusters

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.cluster.GetName() }},
	{Label: "Type", Getter: func(f *finding) string { return f.cluster.GetType().String() }},
	{Label: "Status", Getter: func(f *finding) string {
		return f.cluster.GetHealthStatus().GetOverallHealthStatus().String()
	}},
	{Label: "Upgradability", Getter: func(f *finding) string {
		return f.cluster.GetStatus().GetUpgradeStatus().GetUpgradability().String()
	}},
}

type finding struct {
	tracker.CommonFinding
	cluster *storage.Cluster
	err     error
}
