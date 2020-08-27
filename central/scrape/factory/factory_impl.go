package factory

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
)

type scrapeFactory struct {
	connManager connection.Manager
}

func newFactory(connManager connection.Manager) *scrapeFactory {
	return &scrapeFactory{
		connManager: connManager,
	}
}

func (f *scrapeFactory) RunScrape(domain framework.ComplianceDomain, kill concurrency.Waitable, standardIDs []string) (map[string]*compliance.ComplianceReturn, error) {
	clusterID := domain.Cluster().ID()

	conn := f.connManager.GetConnection(clusterID)
	if conn == nil {
		return nil, fmt.Errorf("could not perform host scrape for cluster %q: no active connection from sensor", clusterID)
	}
	if !conn.HasCapability(centralsensor.ComplianceInNodesCap) {
		return nil, fmt.Errorf("could not perform per-node compliance checks for cluster %q: sensor does not support in-node checks", clusterID)
	}

	expectedHostNames := set.NewStringSet()

	for _, node := range domain.Nodes() {
		expectedHostNames.Add(node.Node().GetName())
	}

	return conn.Scrapes().RunScrape(expectedHostNames, kill, standardIDs)
}
