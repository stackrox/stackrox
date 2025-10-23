package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	policyCategoryStore "github.com/stackrox/rox/central/policycategory/store"
	policyCategoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	"github.com/stackrox/rox/generated/storage"
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
	addDefaults(store)
	ad = New(store, policyCategoryEdgeDS.Singleton())

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

	var existingCategoriesSet set.StringSet
	err = s.Walk(sac.WithAllAccess(context.Background()), func(category *storage.PolicyCategory) error {
		existingCategoriesSet.Add(category.GetName())
		return nil
	}, true)
	utils.CrashOnError(err)

	for _, dc := range defaultCategories {
		if existingCategoriesSet.Contains(dc.GetName()) {
			continue
		}
		if err := s.Upsert(policyCategoryCtx, dc); err != nil {
			utils.CrashOnError(err)
		}
	}
	log.Infof("Loaded %d default policy categories", len(defaultCategories))

}
