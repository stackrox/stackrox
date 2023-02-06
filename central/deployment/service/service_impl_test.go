//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/golang/mock/gomock"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	"github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
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
			var ds datastore.DataStore
			var indexingQ queue.WaitableQueue
			var closer func()
			if env.PostgresDatastoreEnabled.BooleanSetting() {
				ds, closer = setupPostgresDatastore(t)
			} else {
				ds, indexingQ, closer = setupRocksDB(t)
			}
			defer closer()

			for _, deployment := range c.deployments {
				assert.NoError(t, ds.UpsertDeployment(ctx, deployment))
			}

			if !env.PostgresDatastoreEnabled.BooleanSetting() {
				indexingDone := concurrency.NewSignal()
				indexingQ.PushSignal(&indexingDone)
				indexingDone.Wait()
			}

			results, err := ds.Search(ctx, queryForLabels())
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

func setupRocksDB(t *testing.T) (datastore.DataStore, queue.WaitableQueue, func()) {
	rocksDB := rocksdbtest.RocksDBForT(t)
	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	dacky, registry, indexingQ := testDackBoxInstance(t, rocksDB, bleveIndex)
	registry.RegisterWrapper(deploymentDackBox.Bucket, deploymentIndex.Wrapper{})

	ds, err := datastore.GetTestRocksBleveDataStore(t, rocksDB, bleveIndex, dacky, dackboxConcurrency.NewKeyFence())
	require.NoError(t, err)

	closer := func() {
		rocksDB.Close()
	}
	return ds, indexingQ, closer
}

func setupPostgresDatastore(t *testing.T) (datastore.DataStore, func()) {
	testingDB := pgtest.ForT(t)
	pool := testingDB.Pool

	mockCtrl := gomock.NewController(t)
	riskDataStore := riskMocks.NewMockDataStore(mockCtrl)
	ds, err := datastore.New(nil, nil, pool, nil, nil, nil, nil, nil, riskDataStore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	closer := func() {
		pool.Close()
	}
	return ds, closer
}
