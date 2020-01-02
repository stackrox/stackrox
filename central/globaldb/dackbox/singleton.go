package globaldb

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	dackBoxInit sync.Once

	// Bucket specifies the prefix for the id map DackBox tracks and stores in the DB.
	Bucket  = []byte("dackbox_graph")
	duckBox *dackbox.DackBox

	log = logging.LoggerForModule()
)

// GetGlobalDackBox returns the global dackbox.DackBox instance.
func GetGlobalDackBox() *dackbox.DackBox {
	initializeDackBox()
	return duckBox
}

func initializeDackBox() {
	dackBoxInit.Do(func() {
		if features.ManagedDB.Enabled() {
			globaldb.RegisterBucket(Bucket, "Graph Keys")
			var err error
			duckBox, err = dackbox.NewDackBox(globaldb.GetGlobalBadgerDB(), Bucket)
			if err != nil {
				log.Panicf("Could not load stored indices: %v", err)
			}
		}
	})
}
