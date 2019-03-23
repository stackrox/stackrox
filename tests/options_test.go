package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/search/options"
	searchService "github.com/stackrox/rox/central/search/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

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
	for _, v := range searchService.GetGlobalSearchCategories().AsSlice() {
		categories = append(categories, v)
	}

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
