package pruner

import (
	"context"
	"testing"

	scanResult "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	compRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	scanSettingBindingDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suitesDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

type pruneImpl struct {
	integrationDS          compIntegration.DataStore
	scanSettingDS          compScanSetting.DataStore
	scanResultDS           scanResult.DataStore
	compRuleDS             compRuleDS.DataStore
	profileDS              profileDS.DataStore
	scanSettingsBindingsDS scanSettingBindingDS.DataStore
	scanDS                 scanDS.DataStore
	suitesDS               suitesDS.DataStore
}

// Pruner consolidates functionality to clean up orphaned compliance operator data
//
//go:generate mockgen-wrapper
type Pruner interface {
	RemoveComplianceResourcesByCluster(ctx context.Context, clusterID string)
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(integrationDS compIntegration.DataStore, scanSettingDS compScanSetting.DataStore, scanResultDS scanResult.DataStore, compRuleDS compRuleDS.DataStore, profileDS profileDS.DataStore, scanSettingsBindingsDS scanSettingBindingDS.DataStore, scanDS scanDS.DataStore, suitesDS suitesDS.DataStore) Pruner {
	return &pruneImpl{
		integrationDS:          integrationDS,
		scanSettingDS:          scanSettingDS,
		scanResultDS:           scanResultDS,
		compRuleDS:             compRuleDS,
		profileDS:              profileDS,
		scanSettingsBindingsDS: scanSettingsBindingsDS,
		scanDS:                 scanDS,
		suitesDS:               suitesDS,
	}
}

// RemoveComplianceResourcesByCluster removes orphaned compliance operator data for the cluster
func (p *pruneImpl) RemoveComplianceResourcesByCluster(ctx context.Context, clusterID string) {
	// Remove the compliance integrations
	if err := p.integrationDS.RemoveComplianceIntegrationByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete compliance integrations for cluster %q: %v", clusterID, err)
	}

	// Remove any scan configurations for the cluster
	if err := p.scanSettingDS.RemoveClusterFromScanConfig(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan configs for cluster %s: %v", clusterID, err)
	}

	// Remove any scan result for the cluster
	if err := p.scanResultDS.DeleteResultsByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan results for cluster %s: %v", clusterID, err)
	}

	// Remove any rule for the cluster
	if err := p.compRuleDS.DeleteRulesByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete rules for cluster %s: %v", clusterID, err)
	}

	// Remove any profile for the cluster
	if err := p.profileDS.DeleteProfilesByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete profiles for cluster %s: %v", clusterID, err)
	}

	// Remove any scan for the cluster
	if err := p.scanDS.DeleteScanByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scans for cluster %s: %v", clusterID, err)
	}

	// Remove any scan setting for the cluster
	if err := p.scanSettingsBindingsDS.DeleteScanSettingByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan setting bindings for cluster %s: %v", clusterID, err)
	}

	// Remove any scan suite for the cluster
	if err := p.suitesDS.DeleteSuitesByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan suites for cluster %s: %v", clusterID, err)
	}
}

// Testing

// GetTestPruner provides a pruner connected to postgres for testing purposes.
func GetTestPruner(t *testing.T, pool postgres.DB) Pruner {
	scanSettingDS := compScanSetting.GetTestPostgresDataStore(t, pool)
	integrationDS := compIntegration.GetTestPostgresDataStore(t, pool)
	scanResultDS := scanResult.GetTestPostgresDataStore(t, pool)
	compRuleDS := compRuleDS.GetTestPostgresDataStore(t, pool)
	profileDS := profileDS.GetTestPostgresDataStore(t, pool, nil)
	scanSettingsBindingsDS := scanSettingBindingDS.GetTestPostgresDataStore(t, pool)
	scanDS := scanDS.GetTestPostgresDataStore(t, pool)
	suitesDS := suitesDS.GetTestPostgresDataStore(t, pool)
	return New(integrationDS, scanSettingDS, scanResultDS, compRuleDS, profileDS, scanSettingsBindingsDS, scanDS, suitesDS)
}
