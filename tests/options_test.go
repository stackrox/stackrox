package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/search/options"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewSearchServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	categories := []v1.SearchCategory{v1.SearchCategory_ALERTS}
	resp, err := service.Options(ctx, &v1.SearchOptionsRequest{Categories: categories})
	cancel()
	require.NoError(t, err)
	assert.ElementsMatch(t, options.GetOptions(categories), resp.GetOptions())

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	categories = []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS}
	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: categories})
	cancel()
	require.NoError(t, err)
	assert.ElementsMatch(t, options.GetOptions(categories), resp.GetOptions())

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	categories = []v1.SearchCategory{v1.SearchCategory_IMAGES}
	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: categories})
	cancel()
	require.NoError(t, err)
	assert.ElementsMatch(t, options.GetOptions(categories), resp.GetOptions())

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	categories = []v1.SearchCategory{v1.SearchCategory_POLICIES}
	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: categories})
	cancel()
	require.NoError(t, err)
	assert.ElementsMatch(t, options.GetOptions(categories), resp.GetOptions())

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	categories = []v1.SearchCategory{v1.SearchCategory_ALERTS, v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES, v1.SearchCategory_POLICIES, v1.SearchCategory_SECRETS}
	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{Categories: categories})
	cancel()
	require.NoError(t, err)
	assert.ElementsMatch(t, options.GetOptions(categories), resp.GetOptions())

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	resp, err = service.Options(ctx, &v1.SearchOptionsRequest{})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, options.GetOptions(categories), resp.GetOptions())
}
