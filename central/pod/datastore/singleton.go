package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/pod/store/postgres"
	"github.com/stackrox/rox/central/pod/store/rocksdb"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/pkg/features"
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

		if features.PostgresPOC.Enabled() {
			ps, err = New(
				postgres.New(globaldb.GetPostgresDB()),
				globalindex.GetPodIndex(),
				piDS.Singleton(),
				filter.Singleton(),
			)
		} else {
			ps, err = New(
				rocksdb.New(globaldb.GetRocksDB()),
				globalindex.GetPodIndex(),
				piDS.Singleton(),
				filter.Singleton(),
			)
		}
		utils.CrashOnError(errors.Wrap(err, "unable to load datastore for pods"))
	})
	return ps
}
