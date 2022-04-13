package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	piDS "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/central/processindicator/filter"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
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
		if features.PostgresDatastore.Enabled() {
			ps, err = NewPostgresDB(globaldb.GetPostgres(), piDS.Singleton(), filter.Singleton())
		} else {
			ps, err = NewRocksDB(
				globaldb.GetRocksDB(),
				globalindex.GetPodIndex(),
				piDS.Singleton(),
				filter.Singleton(),
			)
		}
		utils.CrashOnError(errors.Wrap(err, "unable to load datastore for pods"))
	})
	return ps
}
