package pruner

import (
	"context"
	"testing"

	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

type pruneImpl struct {
	integrationDS compIntegration.DataStore
	scanSettingDS compScanSetting.DataStore
}

// Pruner consolidates functionality to clean up orphaned compliance operator data
//
//go:generate mockgen-wrapper
type Pruner interface {
	RemoveComplianceResourcesByCluster(ctx context.Context, clusterID string)
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(integrationDS compIntegration.DataStore, scanSettingDS compScanSetting.DataStore) Pruner {
	return &pruneImpl{
		integrationDS: integrationDS,
		scanSettingDS: scanSettingDS,
	}
}

// RemoveComplianceResourcesByCluster removes orphaned compliance operator data for the cluster
func (p *pruneImpl) RemoveComplianceResourcesByCluster(ctx context.Context, clusterID string) {
	// Remove the compliance integrations
	if err := p.integrationDS.RemoveComplianceIntegrationByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete compliance integration for cluster %q: %v", clusterID, err)
	}

	// Remove any scan configurations for the cluster
	if err := p.scanSettingDS.RemoveClusterFromScanConfig(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan config for cluster %s: %v", clusterID, err)
	}
}

// Testing

// GetTestPruner provides a pruner connected to postgres for testing purposes.
func GetTestPruner(t *testing.T, pool postgres.DB) Pruner {
	scanSettingDS := compScanSetting.GetTestPostgresDataStore(t, pool)
	integrationDS := compIntegration.GetTestPostgresDataStore(t, pool)
	return New(integrationDS, scanSettingDS)
}
