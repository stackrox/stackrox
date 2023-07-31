package info

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	installationStore "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/version"
)

// FetchInstallInfo fetches the installation info.
func FetchInstallInfo(ctx context.Context, installation installationStore.Store) *storage.InstallationInfo {
	installInfo, _, err := installation.Get(
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.InstallationInfo),
			),
		),
	)
	if err != nil {
		log.Error("failed to fetch installation information", logging.Err(err))
	}
	return installInfo
}

func getHosting() string {
	if env.ManagedCentral.BooleanSetting() {
		return "cloud-service"
	}
	return "self-managed"
}

func newGaugeVec(installation installationStore.Store) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "info",
			Help:      "A metric with a constant '1' value labeled by information identifying the Central installation",
			ConstLabels: prometheus.Labels{
				"central_id":      FetchInstallInfo(context.Background(), installation).GetId(),
				"central_version": version.GetMainVersion(),
				"hosting":         getHosting(),
				"install_method":  env.InstallMethod.Setting(),
			},
		},
		[]string{"secured_clusters", "secured_nodes", "secured_vcpu"},
	)
}
