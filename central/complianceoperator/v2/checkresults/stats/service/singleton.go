package service

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	benchmarkDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	resultsDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	ruleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	configDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	serviceInstance     Service
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance of the compliance service.
func Singleton() Service {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	serviceInstanceInit.Do(func() {
		serviceInstance = New(resultsDS.Singleton(), configDS.Singleton(), complianceIntegrationDS.Singleton(), profileDatastore.Singleton(), scanDS.Singleton(), benchmarkDS.Singleton(), ruleDS.Singleton(), clusterDatastore.Singleton())
	})
	return serviceInstance
}
