package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/machinebox/graphql"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	graphQLOnce sync.Once

	headerWithBasicAuth http.Header
	graphqlClient       *graphql.Client
)

func makeGraphQLRequest(t *testing.T, query string, vars map[string]interface{}, resp interface{}, timeout time.Duration) {
	graphQLOnce.Do(func() {
		httpReq := http.Request{Header: make(http.Header)}
		httpReq.SetBasicAuth(testutils.RoxUsername(t), testutils.RoxPassword(t))
		headerWithBasicAuth = httpReq.Header

		url, err := urlfmt.FormatURL(testutils.RoxAPIEndpoint(t), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
		require.NoError(t, err)
		graphqlClient = graphql.NewClient(fmt.Sprintf("%s/api/graphql", url),
			graphql.WithHTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}),
		)
	})

	req := graphql.NewRequest(query)
	req.Header = headerWithBasicAuth
	for key, val := range vars {
		req.Var(key, val)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	assert.NoError(t, graphqlClient.Run(ctx, req, resp))
}
