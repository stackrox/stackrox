package tests

import (
	"context"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	graphQLOnce sync.Once

	graphqlClient *graphql.Client
)

func makeGraphQLRequest(t testutils.T, query string, vars map[string]interface{}, resp interface{}, timeout time.Duration) {
	graphQLOnce.Do(func() {
		graphqlClient = graphql.NewClient("/api/graphql", graphql.WithHTTPClient(centralgrpc.HTTPClientForCentral(t)))
		require.NotNil(t, graphqlClient)
	})

	req := graphql.NewRequest(query)
	for key, val := range vars {
		req.Var(key, val)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert.NoError(t, graphqlClient.Run(ctx, req, resp))
}
