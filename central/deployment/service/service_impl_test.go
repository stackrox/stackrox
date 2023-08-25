//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestLabelsMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		deployments    []*storage.Deployment
		expectedMap    map[string]*v1.DeploymentLabelsResponse_LabelValues
		expectedValues []string
	}{
		{
			name: "one deployment",
			deployments: []*storage.Deployment{
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"key": "value",
					},
				},
			},
			expectedMap: map[string]*v1.DeploymentLabelsResponse_LabelValues{
				"key": {
					Values: []string{"value"},
				},
			},
			expectedValues: []string{
				"value",
			},
		},
		{
			name: "multiple deployments",
			deployments: []*storage.Deployment{
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"key":   "value",
						"hello": "world",
						"foo":   "bar",
					},
				},
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"key": "hole",
						"app": "data",
						"foo": "bar",
					},
				},
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"hello": "bob",
						"foo":   "boo",
					},
				},
			},
			expectedMap: map[string]*v1.DeploymentLabelsResponse_LabelValues{
				"key": {
					Values: []string{"hole", "value"},
				},
				"hello": {
					Values: []string{"bob", "world"},
				},
				"foo": {
					Values: []string{"bar", "boo"},
				},
				"app": {
					Values: []string{"data"},
				},
			},
			expectedValues: []string{
				"bar", "bob", "boo", "data", "hole", "value", "world",
			},
		},
	}

	ctx := sac.WithAllAccess(context.Background())
	mockCtrl := gomock.NewController(t)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any()).AnyTimes()
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds, closer := setupPostgresDatastore(t)
			defer closer()

			for _, deployment := range c.deployments {
				assert.NoError(t, ds.UpsertDeployment(ctx, deployment))
			}

			results, err := ds.Search(ctx, queryForLabels())
			assert.NoError(t, err)
			actualMap, actualValues := labelsMapFromSearchResults(results)

			assert.Equal(t, c.expectedMap, actualMap)
			assert.ElementsMatch(t, c.expectedValues, actualValues)
		})
	}
}

func setupPostgresDatastore(t *testing.T) (datastore.DataStore, func()) {
	testingDB := pgtest.ForT(t)
	pool := testingDB.DB

	mockCtrl := gomock.NewController(t)
	riskDataStore := riskMocks.NewMockDataStore(mockCtrl)
	ds, err := datastore.New(pool, nil, nil, nil, riskDataStore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	closer := func() {
		pool.Close()
	}
	return ds, closer
}
