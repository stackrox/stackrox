package datastore

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/search"
	policyStore "github.com/stackrox/rox/central/policy/store"
	categoriesDS "github.com/stackrox/rox/central/policycategory/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := policyStore.New(globaldb.GetPostgres())
	searcher := search.New(storage)

	clusterDatastore := clusterDS.Singleton()
	notifierDatastore := notifierDS.Singleton()
	categoriesDatastore := categoriesDS.Singleton()

	ad = New(storage, searcher, clusterDatastore, notifierDatastore, categoriesDatastore)
	addDefaults(storage, categoriesDatastore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)

	count, _ := ad.Count(
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.WorkflowAdministration),
			),
		), pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PolicySource, storage.PolicySource_DECLARATIVE.String()).ProtoQuery())
	metrics.UpdatePolicyAsCodeCRsReceivedGauge(count)
	return ad
}

// addDefaults adds the default policies into the postgres table for policies.
// TODO: ROX-11279: Data migration for postgres should take care of removing default policies in the bolt bucket named removed_default_policies
// from the policies table in postgres
func addDefaults(s policyStore.Store, categoriesDS categoriesDS.DataStore) {
	policyIDSet := set.NewStringSet()
	storedPolicies, err := s.GetAll(workflowAdministrationCtx)
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
		p.Categories = []string{}
		if err := s.Upsert(workflowAdministrationCtx, p); err != nil {
			utils.Must(err)
		}
		if err := categoriesDS.SetPolicyCategoriesForPolicy(sac.WithAllAccess(context.Background()), p.GetId(), policyCategories); err != nil {
			utils.Should(err)
		}

	}
	log.Infof("Loaded %d new default Policies", count)
}
