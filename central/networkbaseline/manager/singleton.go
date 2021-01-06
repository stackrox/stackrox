package manager

import (
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	once.Do(func() {
		var err error
		instance, err = New(datastore.Singleton(), networkEntityDS.Singleton())
		utils.Must(err)
	})
	return instance
}
