package datastore

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	policyStore "github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/central/policy/store/boltdb"
	policyPostgres "github.com/stackrox/rox/central/policy/store/postgres"
	categoriesDS "github.com/stackrox/rox/central/policycategory/datastore"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/policyutils"
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
	var storage policyStore.Store
	var indexer index.Indexer

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = policyPostgres.New(globaldb.GetPostgres())
		indexer = policyPostgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = boltdb.New(globaldb.GetGlobalDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}

	searcher := search.New(storage, indexer)

	clusterDatastore := clusterDS.Singleton()
	notifierDatastore := notifierDS.Singleton()
	categoriesDatastore := categoriesDS.Singleton()

	ad = New(storage, indexer, searcher, clusterDatastore, notifierDatastore, categoriesDatastore)
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		addDefaults(storage, categoriesDatastore)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

// addDefaults adds the default policies into the postgres table for policies.
// TODO: ROX-11279: Data migration for postgres should take care of removing default policies in the bolt bucket named removed_default_policies
// from the policies table in postgres
func addDefaults(s policyStore.Store, categoriesDS categoriesDS.DataStore) {
	policyIDSet := set.NewStringSet()
	storedPolicies, err := s.GetAll(policyCtx)
	if err != nil {
		panic(err)
	}

	for _, p := range storedPolicies {
		policyIDSet.Add(p.GetId())
	}

	// Preload the default policies.
	defaultPolicies, err := policies.DefaultPolicies()
	// Hard panic here is okay, since we can always guarantee that we will be able to get the default policies out.
	utils.CrashOnError(err)

	var count int
	for _, p := range defaultPolicies {
		// If ID is not the same as the shipped default policy, we treat it as custom policy. Hence, the tombstone
		// state is not tracked.
		if policyIDSet.Contains(p.GetId()) {
			continue
		}
		count++

		// fill multi-word sort helper field
		policyutils.FillSortHelperFields(p)

		policyCategories := p.GetCategories()
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			p.Categories = []string{}
		}
		if err := s.Upsert(policyCtx, p); err != nil {
			utils.CrashOnError(err)
		}
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			if err := categoriesDS.SetPolicyCategoriesForPolicy(sac.WithAllAccess(context.Background()), p.GetId(), policyCategories); err != nil {
				utils.CrashOnError(err)
			}
		}

	}
	log.Infof("Loaded %d new default Policies", count)
}
