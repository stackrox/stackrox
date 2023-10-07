package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	plopDS "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ps DataStore

	log = logging.LoggerForModule()
)

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(func() {
		var err error
		ps, err = NewPostgresDB(globaldb.GetPostgres(), piDS.Singleton(), plopDS.Singleton(), filter.Singleton())
		utils.CrashOnError(errors.Wrap(err, "unable to load datastore for pods"))
	})
	return ps
}
