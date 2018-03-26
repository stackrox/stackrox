package tests

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewSearchServiceClient(conn)
	resp, err := service.Options(ctx, &v1.SearchOptionsRequest{Categories: []v1.SearchCategory{v1.SearchCategory_ALERTS}})
	require.NoError(t, err)
	assert.Equal(t, len(search.AlertOptionsMap)+len(search.DeploymentOptionsMap)+len(search.ImageOptionsMap)+len(search.PolicyOptionsMap)+len(search.GlobalOptions), len(resp.GetOptions()))

	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS}})
	require.NoError(t, err)
	assert.Equal(t, len(search.DeploymentOptionsMap)+len(search.ImageOptionsMap)+len(search.GlobalOptions), len(resp.GetOptions()))

	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: []v1.SearchCategory{v1.SearchCategory_IMAGES}})
	require.NoError(t, err)
	assert.Equal(t, len(search.ImageOptionsMap)+len(search.GlobalOptions), len(resp.GetOptions()))

	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: []v1.SearchCategory{v1.SearchCategory_POLICIES}})
	require.NoError(t, err)
	assert.Equal(t, len(search.PolicyOptionsMap)+len(search.GlobalOptions), len(resp.GetOptions()))

	globalLen := len(search.ImageOptionsMap) + len(search.DeploymentOptionsMap) + len(search.PolicyOptionsMap) + len(search.AlertOptionsMap) + len(search.GlobalOptions)
	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: []v1.SearchCategory{v1.SearchCategory_ALERTS, v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES, v1.SearchCategory_POLICIES}})
	require.NoError(t, err)
	assert.Equal(t, globalLen, len(resp.GetOptions()))

	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{})
	require.NoError(t, err)
	assert.Equal(t, globalLen, len(resp.GetOptions()))
}
