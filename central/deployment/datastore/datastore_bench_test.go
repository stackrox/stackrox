package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	search2 "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllDeployments(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	deploymentsDatastore, err := GetTestPostgresDataStore(b, testDB.DB)
	require.NoError(b, err)

	deploymentPrototype := fixtures.GetDeployment().Clone()
	const numDeployments = 1000
	for i := 0; i < numDeployments; i++ {
		if i > 0 && i%100 == 0 {
			fmt.Println("Added", i, "deployments")
		}
		deploymentPrototype.Id = uuid.NewV4().String()
		require.NoError(b, deploymentsDatastore.UpsertDeployment(ctx, deploymentPrototype))
	}

	b.Run("SearchRetrievalList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.SearchListDeployments(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})

	b.Run("SearchRetrievalFull", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.SearchRawDeployments(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})
}
