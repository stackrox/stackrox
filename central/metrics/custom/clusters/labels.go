package clusters

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var LazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.GetName() }},
	{Label: "Type", Getter: func(f *finding) string { return f.GetType().String() }},
	{Label: "Status", Getter: func(f *finding) string {
		return f.GetHealthStatus().GetOverallHealthStatus().String()
	}},
	{Label: "Upgradability", Getter: func(f *finding) string {
		return f.GetStatus().GetUpgradeStatus().GetUpgradability().String()
	}},
}

type finding struct {
	*storage.Cluster
}
