package factory

import (
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	factoryInstance     ScrapeFactory
	factoryInstanceInit sync.Once
)

// Singleton returns the singleton instance for the scrape factory.
func Singleton() ScrapeFactory {
	factoryInstanceInit.Do(func() {
		factoryInstance = newFactory(connection.ManagerSingleton())
	})
	return factoryInstance
}
