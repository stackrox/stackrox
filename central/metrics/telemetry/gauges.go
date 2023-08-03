package telemetry

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	installationStore "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

var (
	securedClustersGaugeName = "secured_clusters"
	securedNodesGaugeName    = "secured_nodes"
	securedVCPUGaugeName     = "secured_vcpu"
)

// FetchInstallInfo fetches the installation info.
//
// If an error is returned, the returned installInfo is nil and the
// install ID would be empty. As a result, the `central_id` label
// would be dropped by Prometheus.
func FetchInstallInfo(ctx context.Context, installation installationStore.Store) (*storage.InstallationInfo, error) {
	installInfo, _, err := installation.Get(
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.InstallationInfo),
			),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch installation information")
	}
	return installInfo, nil
}

func getHosting() string {
	if env.ManagedCentral.BooleanSetting() {
		return "cloud-service"
	}
	return "self-managed"
}

func newGaugeMap(installation installationStore.Store) map[string]prometheus.Gauge {
	installInfo, err := FetchInstallInfo(context.Background(), installation)
	utils.Should(err)
	labels := prometheus.Labels{
		"branding":        branding.GetProductNameShort(),
		"build":           buildinfo.BuildFlavor,
		"central_id":      installInfo.GetId(),
		"central_version": version.GetMainVersion(),
		"hosting":         getHosting(),
		"install_method":  env.InstallMethod.Setting(),
	}

	gauges := map[string]prometheus.Gauge{
		securedClustersGaugeName: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   metrics.PrometheusNamespace,
				Subsystem:   metrics.CentralSubsystem.String(),
				Name:        securedClustersGaugeName,
				Help:        "The number of clusters secured by Central",
				ConstLabels: labels,
			},
		),
		securedNodesGaugeName: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   metrics.PrometheusNamespace,
				Subsystem:   metrics.CentralSubsystem.String(),
				Name:        securedNodesGaugeName,
				Help:        "The number of nodes secured by Central",
				ConstLabels: labels,
			},
		),
		securedVCPUGaugeName: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   metrics.PrometheusNamespace,
				Subsystem:   metrics.CentralSubsystem.String(),
				Name:        securedVCPUGaugeName,
				Help:        "The number of vCPUs secured by Central",
				ConstLabels: labels,
			},
		),
	}
	return gauges
}
