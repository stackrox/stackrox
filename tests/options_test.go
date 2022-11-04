package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/search"
	"github.com/stackrox/rox/central/search/options"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsMapExist(t *testing.T) {
	t.Parallel()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewSearchServiceClient(conn)

	for _, categories := range [][]v1.SearchCategory{
		{},
		{v1.SearchCategory_ALERTS},
		{v1.SearchCategory_DEPLOYMENTS},
		{v1.SearchCategory_IMAGES},
		{v1.SearchCategory_POLICIES},
		search.GetGlobalSearchCategories().AsSlice(),
	} {
		t.Run(fmt.Sprintf("%v", categories), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			resp, err := service.Options(ctx, &v1.SearchOptionsRequest{Categories: categories})
			cancel()
			require.NoError(t, err)
			assert.ElementsMatch(t, options.GetOptions(categories), resp.GetOptions())
		})
	}
}

func TestOptionsMap(t *testing.T) {
	optionsMap := options.GetOptions([]v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
	assert.Contains(t, optionsMap, "Namespace")
}
