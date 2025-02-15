package resolvers

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestClustersForPermissionsWithMock(t *testing.T) {
	storeInvalidErr := errox.InvalidArgs.CausedBy("Wrong arguments")

	testCases := map[string]struct {
		ctx            context.Context
		targetResource permissions.ResourceMetadata

		mockStoreValues []*storage.Cluster
		mockStoreError  error

		expectedResolverValues []*v1.ScopeObject
		expectedResolverError  error
	}{
		"Full Access, Store error": {
			ctx:                    sac.WithAllAccess(context.Background()),
			targetResource:         resources.Compliance,
			mockStoreValues:        nil,
			mockStoreError:         storeInvalidErr,
			expectedResolverValues: nil,
			expectedResolverError:  storeInvalidErr,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			clusterStore := clusterMocks.NewMockDataStore(mockCtrl)
			mainResolver := &Resolver{ClusterDataStore: clusterStore}

			clusterStore.EXPECT().
				SearchRawClusters(gomock.Any(), gomock.Any()).
				Times(1).
				Return(testCase.mockStoreValues, testCase.mockStoreError)

			ctx := testCase.ctx
			query := PaginatedQuery{}
			targetResource := testCase.targetResource

			fetchedClusterResolvers, err := mainResolver.clustersForReadPermission(ctx, query, targetResource)
			if testCase.expectedResolverError != nil {
				assert.ErrorIs(it, err, testCase.expectedResolverError)
				assert.Nil(it, fetchedClusterResolvers)
			} else {
				assert.NoError(it, err)
				fetchedScopeObjects := make([]*v1.ScopeObject, 0, len(fetchedClusterResolvers))
				for _, objectResolver := range fetchedClusterResolvers {
					if objectResolver == nil {
						continue
					} else {
						fetchedScopeObjects = append(fetchedScopeObjects, objectResolver.data)
					}
				}
				protoassert.ElementsMatch(t, testCase.expectedResolverValues, fetchedScopeObjects)
			}
		})
	}
}
