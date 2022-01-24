package enrichment

import (
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/cve/fetcher"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/central/vulnerabilityrequest/suppressor"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/metrics"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ie      imageEnricher.ImageEnricher
	ne      nodeEnricher.NodeEnricher
	en      Enricher
	cf      fetcher.OrchestratorIstioCVEManager
	manager Manager
)

func initialize() {
	ie = imageEnricher.New(cveDataStore.Singleton(), suppressor.Singleton(), imageintegration.Set(), metrics.CentralSubsystem, datastore.Singleton().GetImage, reporter.Singleton())
	ne = nodeEnricher.New(cveDataStore.Singleton(), metrics.CentralSubsystem)
	en = New(datastore.Singleton(), ie)
	cf = fetcher.SingletonManager()
	manager = newManager(imageintegration.Set(), ne, cf)
}

// Singleton provides the singleton Enricher to use.
func Singleton() Enricher {
	once.Do(initialize)
	return en
}

// ImageEnricherSingleton provides the singleton ImageEnricher to use.
func ImageEnricherSingleton() imageEnricher.ImageEnricher {
	once.Do(initialize)
	return ie
}

// NodeEnricherSingleton provides the singleton NodeEnricher to use.
func NodeEnricherSingleton() nodeEnricher.NodeEnricher {
	once.Do(initialize)
	return ne
}

// ManagerSingleton returns the multiplexing manager
func ManagerSingleton() Manager {
	once.Do(initialize)
	return manager
}
