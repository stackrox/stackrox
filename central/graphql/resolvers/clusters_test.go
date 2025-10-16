package resolvers

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestClustersForPermissions(t *testing.T) {
	cluster1 := &storage.Cluster{}
	cluster1.SetId(fixtureconsts.Cluster1)
	cluster1.SetName("Cluster 1")
	scopeObject1 := &v1.ScopeObject{}
	scopeObject1.SetId(fixtureconsts.Cluster1)
	scopeObject1.SetName("Cluster 1")
	cluster2 := &storage.Cluster{}
	cluster2.SetId(fixtureconsts.Cluster2)
	cluster2.SetName("Cluster 2")
	scopeObject2 := &v1.ScopeObject{}
	scopeObject2.SetId(fixtureconsts.Cluster2)
	scopeObject2.SetName("Cluster 2")
	cluster3 := &storage.Cluster{}
	cluster3.SetId(fixtureconsts.Cluster3)
	cluster3.SetName("Cluster 3")
	scopeObject3 := &v1.ScopeObject{}
	scopeObject3.SetId(fixtureconsts.Cluster3)
	scopeObject3.SetName("Cluster 3")
	storeInvalidErr := errox.InvalidArgs.CausedBy("Wrong arguments")

	testCases := map[string]struct {
		ctx            context.Context
		targetResource permissions.ResourceMetadata

		expectedStoreValues []*storage.Cluster
		expectedStoreError  error

		expectedResolverValues []*v1.ScopeObject
		expectedResolverError  error
	}{
		"Full Access, All cluster retrieved": {
			ctx:                    sac.WithAllAccess(context.Background()),
			targetResource:         resources.Compliance,
			expectedStoreValues:    []*storage.Cluster{cluster1, cluster2, cluster3},
			expectedStoreError:     nil,
			expectedResolverValues: []*v1.ScopeObject{scopeObject1, scopeObject2, scopeObject3},
			expectedResolverError:  nil,
		},
		"Full Access, Store error": {
			ctx:                    sac.WithAllAccess(context.Background()),
			targetResource:         resources.Compliance,
			expectedStoreValues:    nil,
			expectedStoreError:     storeInvalidErr,
			expectedResolverValues: nil,
			expectedResolverError:  storeInvalidErr,
		},
		"Unrestricted Read on target resource, All clusters retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Compliance),
				),
			),
			targetResource:         resources.Compliance,
			expectedStoreValues:    []*storage.Cluster{cluster1, cluster2, cluster3},
			expectedStoreError:     nil,
			expectedResolverValues: []*v1.ScopeObject{scopeObject1, scopeObject2, scopeObject3},
			expectedResolverError:  nil,
		},
		"Partial Read on target resource, All allowed clusters retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Compliance),
					sac.ClusterScopeKeys(fixtureconsts.Cluster1),
				),
			),
			targetResource:         resources.Compliance,
			expectedStoreValues:    []*storage.Cluster{cluster1},
			expectedStoreError:     nil,
			expectedResolverValues: []*v1.ScopeObject{scopeObject1},
			expectedResolverError:  nil,
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
				Return(testCase.expectedStoreValues, testCase.expectedStoreError)

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

	t.Run("Unrestricted Read on wrong resource, No cluster retrieved", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		clusterStore := clusterMocks.NewMockDataStore(mockCtrl)
		mainResolver := &Resolver{ClusterDataStore: clusterStore}

		ctx := sac.WithGlobalAccessScopeChecker(
			context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
			),
		)
		query := PaginatedQuery{}
		targetResource := resources.Compliance
		fetchedClusterResolvers, err := mainResolver.clustersForReadPermission(ctx, query, targetResource)
		assert.NoError(it, err)
		assert.Empty(it, fetchedClusterResolvers)
	})
}
