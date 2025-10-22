package clusters

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var LazyLabels = tracker.LazyLabelGetters[*finding]{
	"Cluster": func(f *finding) string { return f.GetName() },
	"Type":    func(f *finding) string { return f.GetType().String() },
	"Status": func(f *finding) string {
		return f.GetHealthStatus().GetOverallHealthStatus().String()
	},
	"Upgradability": func(f *finding) string {
		return f.GetStatus().GetUpgradeStatus().GetUpgradability().String()
	},
}

type finding = storage.Cluster
