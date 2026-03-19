package enricher

import (
	imageCVEInfoDS "github.com/stackrox/rox/central/cve/image/info/datastore"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	instance imageEnricher.CVEInfoEnricher
)

func initialize() {
	instance = New(imageCVEInfoDS.Singleton())
}

// Singleton returns a singleton instance of CVEInfoEnricher.
func Singleton() imageEnricher.CVEInfoEnricher {
	once.Do(initialize)
	return instance
}
