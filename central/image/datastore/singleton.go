package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	dackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error
	ad, err = NewBadger(dackbox.GetGlobalDackBox(), dackbox.GetKeyFence(), globaldb.GetGlobalBadgerDB(), globalindex.GetGlobalIndex(), false, imageComponentDS.Singleton(), riskDS.Singleton(), ranking.ImageRanker())
	utils.Must(errors.Wrap(err, "unable to load datastore for images"))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
