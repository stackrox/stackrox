//go:build sql_integration

package postgres

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func TestManyClusterSACFiltersExecute(t *testing.T) {
	t.Setenv(env.PostgresParameterThreshold.EnvVar(), "100")
	db := pgtest.ForT(t)
	ids, clusters := namespaceScopeFixture(700, 99)
	for _, deployment := range []struct{ cluster, namespace string }{
		{ids[0], "namespace-0"},
		{ids[0], "namespace-98"},
		{ids[0], "namespace-1"},
		{ids[0], "namespace-99"},
		{uuid.NewV4().String(), "namespace-0"},
	} {
		_, err := db.Exec(t.Context(), "INSERT INTO deployments (id, clusterid, namespace) VALUES ($1, $2, $3)",
			uuid.NewV4().String(), deployment.cluster, deployment.namespace)
		require.NoError(t, err)
	}
	ctx := namespaceScopeContext(t, clusters)
	count, err := RunCountRequestForSchema(ctx, schema.DeploymentsSchema, search.EmptyQuery(), db)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	wholeClusters := map[string]*sac.TestClusterScope{
		ids[0]: {Included: true},
		ids[1]: {Included: true},
	}
	count, err = RunCountRequestForSchema(namespaceScopeContext(t, wholeClusters), schema.DeploymentsSchema,
		search.EmptyQuery(), db)
	require.NoError(t, err)
	require.Equal(t, 4, count)

	count, err = RunCountRequestForSchema(sac.WithAllAccess(t.Context()), schema.DeploymentsSchema,
		namespaceExclusionQuery(ids, clusters), db)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
