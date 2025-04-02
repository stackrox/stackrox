package pruner

import (
	"context"
	"testing"

	scanResult "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profile "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	compRule "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scan "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	scanSettingBinding "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suites "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

type pruneImpl struct {
	integrationDataStore          compIntegration.DataStore
	scanSettingDataStore          compScanSetting.DataStore
	scanResultDataStore           scanResult.DataStore
	compRuleDataStore             compRule.DataStore
	profileDataStore              profile.DataStore
	scanSettingsBindingsDataStore scanSettingBinding.DataStore
	scanDataStore                 scan.DataStore
	suitesDataStore               suites.DataStore
}

// Pruner consolidates functionality to clean up orphaned compliance operator data
//
//go:generate mockgen-wrapper
type Pruner interface {
	RemoveComplianceResourcesByCluster(ctx context.Context, clusterID string)
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(integrationDataStore compIntegration.DataStore, scanSettingDataStore compScanSetting.DataStore, scanResultDataStore scanResult.DataStore, compRuleDataStore compRule.DataStore, profileDataStore profile.DataStore, scanSettingsBindingsDataStore scanSettingBinding.DataStore, scanDataStore scan.DataStore, suitesDataStore suites.DataStore) Pruner {
	return &pruneImpl{
		integrationDataStore:          integrationDataStore,
		scanSettingDataStore:          scanSettingDataStore,
		scanResultDataStore:           scanResultDataStore,
		compRuleDataStore:             compRuleDataStore,
		profileDataStore:              profileDataStore,
		scanSettingsBindingsDataStore: scanSettingsBindingsDataStore,
		scanDataStore:                 scanDataStore,
		suitesDataStore:               suitesDataStore,
	}
}

// RemoveComplianceResourcesByCluster removes orphaned compliance operator data for the cluster
func (p *pruneImpl) RemoveComplianceResourcesByCluster(ctx context.Context, clusterID string) {
	// Remove the compliance integrations
	if err := p.integrationDataStore.RemoveComplianceIntegrationByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete compliance integrations for cluster %q: %v", clusterID, err)
	}

	// Remove any scan configurations for the cluster
	if err := p.scanSettingDataStore.RemoveClusterFromScanConfig(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan configs for cluster %s: %v", clusterID, err)
	}

	// Remove any scan result for the cluster
	if err := p.scanResultDataStore.DeleteResultsByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan results for cluster %s: %v", clusterID, err)
	}

	// Remove any rule for the cluster
	if err := p.compRuleDataStore.DeleteRulesByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete rules for cluster %s: %v", clusterID, err)
	}

	// Remove any profile for the cluster
	if err := p.profileDataStore.DeleteProfilesByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete profiles for cluster %s: %v", clusterID, err)
	}

	// Remove any scan for the cluster
	if err := p.scanDataStore.DeleteScanByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scans for cluster %s: %v", clusterID, err)
	}

	// Remove any scan setting for the cluster
	if err := p.scanSettingsBindingsDataStore.DeleteScanSettingByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan setting bindings for cluster %s: %v", clusterID, err)
	}

	// Remove any scan suite for the cluster
	if err := p.suitesDataStore.DeleteSuitesByCluster(ctx, clusterID); err != nil {
		log.Errorf("failed to delete scan suites for cluster %s: %v", clusterID, err)
	}
}

// Testing

// GetTestPruner provides a pruner connected to postgres for testing purposes.
func GetTestPruner(t testing.TB, pool postgres.DB) Pruner {
	scanSettingDataStore := compScanSetting.GetTestPostgresDataStore(t, pool)
	integrationDataStore := compIntegration.GetTestPostgresDataStore(t, pool)
	scanResultDataStore := scanResult.GetTestPostgresDataStore(t, pool)
	compRuleDataStore := compRule.GetTestPostgresDataStore(t, pool)
	profileDataStore := profile.GetTestPostgresDataStore(t, pool, nil)
	scanSettingsBindingsDataStore := scanSettingBinding.GetTestPostgresDataStore(t, pool)
	scanDataStore := scan.GetTestPostgresDataStore(t, pool)
	suitesDataStore := suites.GetTestPostgresDataStore(t, pool)

	return New(integrationDataStore, scanSettingDataStore, scanResultDataStore, compRuleDataStore, profileDataStore, scanSettingsBindingsDataStore, scanDataStore, suitesDataStore)
}
