package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"

	policyStore "github.com/stackrox/rox/central/policy/store"

	policyCategoryDS "github.com/stackrox/rox/central/policycategory/datastore"
	policyCategoryStore "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	policyCategoryEdgeStore "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
)

func BenchmarkGetAllPolcies(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	b.Setenv("CI", "true")
	b.Setenv("POSTGRES_PASSWORD", "postgres")
	testDB := pgtest.ForT(b)
	defer testDB.Close()

	db := testDB.DB
	defer db.Close()
	gormDB := testDB.GetGormDB(b)
	defer pgtest.CloseGormDB(b, gormDB)

	edgeStore := policyCategoryEdgeStore.New(db)
	edgeDS := policyCategoryEdgeDS.New(edgeStore)

	categoryStore := policyCategoryStore.New(db)
	categoryDS := policyCategoryDS.New(categoryStore, edgeDS)

	storage := policyStore.New(db)
	policyDS := New(storage, nil, nil, categoryDS)
	seedPolicies(b, ctx, 100, policyDS)

	for i := 0; i < b.N; i++ {
		policyDS.GetAllPolicies(ctx)
	}

}

func seedPolicies(t *testing.B, ctx context.Context, count int, ds DataStore) {
	categories := []string{"Security", "DevOps", "Compliance", "Network"}

	for i := 0; i < count; i++ {
		policy := &storage.Policy{
			Name:            fmt.Sprintf("Policy %d", i),
			Description:     "Test policy",
			Severity:        storage.Severity_LOW_SEVERITY,
			Categories:      []string{categories[i%len(categories)]},
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: "Image Remote",
							Values:    []*storage.PolicyValue{{Value: ".*"}},
						},
					},
				},
			},
		}

		_, err := ds.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

}
