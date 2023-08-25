package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/policycategory/search"
	policyCategoryStore "github.com/stackrox/rox/central/policycategory/store"
	policyCategoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	"github.com/stackrox/rox/pkg/defaults/categories"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	store := policyCategoryPostgres.New(globaldb.GetPostgres())
	indexer := policyCategoryPostgres.NewIndexer(globaldb.GetPostgres())
	addDefaults(store)
	searcher := search.New(store, indexer)
	ad = New(store, searcher, policyCategoryEdgeDS.Singleton())

}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

func addDefaults(s policyCategoryStore.Store) {
	// Preload the default policies.
	defaultCategories, err := categories.DefaultPolicyCategories()
	// Hard panic here is okay, since we can always guarantee that we will be able to get the default policy categories out.
	utils.CrashOnError(err)

	existingCategories, err := s.GetAll(sac.WithAllAccess(context.Background()))
	utils.CrashOnError(err)

	var existingCategoriesSet set.StringSet
	for _, category := range existingCategories {
		existingCategoriesSet.Add(category.GetName())
	}

	for _, dc := range defaultCategories {
		if existingCategoriesSet.Contains(dc.Name) {
			continue
		}
		if err := s.Upsert(policyCategoryCtx, dc); err != nil {
			utils.CrashOnError(err)
		}
	}
	log.Infof("Loaded %d default policy categories", len(defaultCategories))

}
