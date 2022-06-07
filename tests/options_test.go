package tests

import (
	"context"
	"testing"
	"time"

	search "github.com/stackrox/rox/central/search"
	"github.com/stackrox/rox/central/search/options"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	t.Parallel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

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

	// All category

	categories = categories[:0]
	categories = append(categories, search.GetGlobalSearchCategories().AsSlice()...)

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
