package datastore

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	policyStore "github.com/stackrox/rox/central/policy/store"
	categoriesDS "github.com/stackrox/rox/central/policycategory/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := policyStore.New(globaldb.GetPostgres())

	clusterDatastore := clusterDS.Singleton()
	notifierDatastore := notifierDS.Singleton()
	categoriesDatastore := categoriesDS.Singleton()

	ad = New(storage, clusterDatastore, notifierDatastore, categoriesDatastore)
	addDefaults(storage, categoriesDatastore, ad)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

// addDefaults adds the default policies into the postgres table for policies.
// TODO: ROX-11279: Data migration for postgres should take care of removing default policies in the bolt bucket named removed_default_policies
// from the policies table in postgres
func addDefaults(s policyStore.Store, categoriesDS categoriesDS.DataStore, fullStore DataStore) {
	policyIDSet := set.NewStringSet()
	storedPolicies := make([]*storage.Policy, 0)
	err := s.Walk(workflowAdministrationCtx, func(p *storage.Policy) error {
		policyIDSet.Add(p.GetId())
		// Unrelated to adding/checking default policies, this was put here to prevent looping through all policies a second time
		if p.GetSource() == storage.PolicySource_DECLARATIVE {
			metrics.IncrementTotalExternalPoliciesGauge()
		}
		storedPolicies = append(storedPolicies, p)
		return nil
	})

	if err != nil {
		panic(err)
	}

	// ROX-31406: Fix categories that were impacted by previous bug
	for _, p := range storedPolicies {
		var categories []*storage.PolicyCategory
		categories, err = categoriesDS.GetPolicyCategoriesForPolicy(workflowAdministrationCtx, p.GetId())
		if err != nil {
			panic(err)
		}
		shouldReupsert := false
		p.Categories = sliceutils.Map[*storage.PolicyCategory, string](categories, func(c *storage.PolicyCategory) string {
			// Both Docker CIS and DevOps Best Practices were broken as a result of a change made in 4.8 that added
			// a title case enforcement on policies that were added, not accounting for the fact that words may have
			// more than just the first character capitalized. This code section just fixes the default categories
			// that may have been duplicated as a result of this.
			if c.GetName() == "Docker Cis" {
				shouldReupsert = true
				return "Docker CIS"
			} else if c.GetName() == "Devops Best Practices" {
				shouldReupsert = true
				return "DevOps Best Practices"
			}
			return c.GetName()
		})
		if shouldReupsert {
			// Update policy, taking advantage of the full datastore updating edges for us
			err = fullStore.UpdatePolicy(sac.WithAllAccess(context.Background()), p)
			if err != nil {
				panic(err)
			}
		}
	}

	// Clean up invalid policy categories
	var results []searchPkg.Result
	q := searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.PolicyCategoryName, "Devops Best Practices", "Docker Cis").ProtoQuery()
	results, err = categoriesDS.Search(workflowAdministrationCtx, q)
	if err != nil {
		panic(err)
	}
	for _, result := range results {
		err = categoriesDS.DeletePolicyCategory(sac.WithAllAccess(context.Background()), result.ID)
		if err != nil {
			panic(err)
		}
	}
	// End ROX-31406-specific code

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
