package service

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	deploymentDackBox "github.com/stackrox/stackrox/central/deployment/dackbox"
	"github.com/stackrox/stackrox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/stackrox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/indexer"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	"github.com/stackrox/stackrox/pkg/grpc/testutils"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			rocksDB := rocksdbtest.RocksDBForT(t)
			defer rocksDB.Close()

			bleveIndex, err := globalindex.MemOnlyIndex()
			require.NoError(t, err)

			dacky, registry, indexingQ := testDackBoxInstance(t, rocksDB, bleveIndex)
			registry.RegisterWrapper(deploymentDackBox.Bucket, deploymentIndex.Wrapper{})

			deploymentsDS := datastore.New(dacky, concurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, nil, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

			for _, deployment := range c.deployments {
				assert.NoError(t, deploymentsDS.UpsertDeployment(ctx, deployment))
			}

			indexingDone := concurrency.NewSignal()
			indexingQ.PushSignal(&indexingDone)
			indexingDone.Wait()

			results, err := deploymentsDS.Search(ctx, queryForLabels())
			assert.NoError(t, err)
			actualMap, actualValues := labelsMapFromSearchResults(results)

			assert.Equal(t, c.expectedMap, actualMap)
			assert.ElementsMatch(t, c.expectedValues, actualValues)
		})
	}
}

func testDackBoxInstance(t *testing.T, db *rocksdb.RocksDB, index bleve.Index) (*dackbox.DackBox, indexer.WrapperRegistry, queue.WaitableQueue) {
	indexingQ := queue.NewWaitableQueue()
	dacky, err := dackbox.NewRocksDBDackBox(db, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	reg := indexer.NewWrapperRegistry()
	lazy := indexer.NewLazy(indexingQ, reg, index, dacky.AckIndexed)
	lazy.Start()

	return dacky, reg, indexingQ
}
