package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error

	if features.BadgerDB.Enabled() {
		ad, err = NewBadger(globaldb.GetGlobalBadgerDB(), globalindex.GetGlobalIndex(), false, riskDS.Singleton())
	} else {
		ad, err = NewBolt(globaldb.GetGlobalDB(), globalindex.GetGlobalIndex(), false, riskDS.Singleton())
	}
	utils.Must(errors.Wrap(err, "unable to load datastore for images"))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
