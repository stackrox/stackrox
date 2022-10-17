//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/heimdalr/dag"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestCollectionDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(CollectionPostgresDataStoreTestSuite))
}

type CollectionPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx         context.Context
	db          *pgxpool.Pool
	gormDB      *gorm.DB
	store       postgres.Store
	datastore   DataStore
	envIsolator *envisolator.EnvIsolator
}

func (s *CollectionPostgresDataStoreTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.db = pool

	postgres.Destroy(s.ctx, s.db)

	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.store = postgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	index := postgres.NewIndexer(s.db)
	s.datastore = New(s.store, index, search.New(s.store, index))
}

// SetupTest removes the local graph before every test
func (s *CollectionPostgresDataStoreTestSuite) SetupTest() {
	s.NoError(resetLocalGraph(s.datastore.(*datastoreImpl)))
}

func (s *CollectionPostgresDataStoreTestSuite) TearDownSuite() {
	postgres.Destroy(s.ctx, s.db)
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
	s.envIsolator.RestoreAll()
}

func (s *CollectionPostgresDataStoreTestSuite) TestGraphInit() {
	ctx := sac.WithAllAccess(context.Background())

	for _, tc := range []struct {
		desc string
		size int
	}{
		{
			desc: "Test Graph Init small",
			size: 2,
		},
		{
			desc: "Test Graph Init initBatchSize-1",
			size: initBatchSize - 1,
		},
		{
			desc: "Test Graph Init initBatchSize",
			size: initBatchSize,
		},
		{
			desc: "Test Graph Init initBatchSize+1",
			size: initBatchSize + 1,
		},
		{
			desc: "Test Graph Init initBatchSize+2",
			size: initBatchSize + 2,
		},
		{
			desc: "Test Graph Init initBatchSize*2",
			size: initBatchSize * 2,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			objs := make([]*storage.ResourceCollection, 0, tc.size)
			objIDs := make([]string, 0, tc.size+1)

			objs = append(objs, s.getTestCollection("0", nil))
			objIDs = append(objIDs, "0")
			for i := 1; i < tc.size; i++ {
				edges := make([]string, 0, i)
				for j := 0; j < i; j++ {
					edges = append(edges, fmt.Sprintf("%d", j))
				}
				id := fmt.Sprintf("%d", i)
				objs = append(objs, s.getTestCollection(id, edges))
				objIDs = append(objIDs, id)
			}

			// add objs directly through the store
			err := s.store.UpsertMany(ctx, objs)
			assert.NoError(s.T(), err)

			// trigger graph init
			err = resetLocalGraph(s.datastore.(*datastoreImpl))
			assert.NoError(s.T(), err)

			// get data and check it
			batch, err := s.datastore.GetBatch(ctx, objIDs)
			assert.NoError(s.T(), err)
			assert.ElementsMatch(s.T(), objs, batch)

			// clean up data
			for i := len(objIDs) - 1; i >= 0; i-- {
				assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objIDs[i]))
			}
			assert.NoError(s.T(), resetLocalGraph(s.datastore.(*datastoreImpl)))
		})
	}
}

func (s *CollectionPostgresDataStoreTestSuite) TestCollectionWorkflows() {
	ctx := sac.WithAllAccess(context.Background())

	// dryrun 'a', verify not present
	err := s.datastore.DryRunCollection(ctx, s.getTestCollection("a", nil))
	assert.NoError(s.T(), err)
	obj, ok, err := s.datastore.Get(ctx, "a")
	assert.NoError(s.T(), err)
	assert.False(s.T(), ok)
	assert.Nil(s.T(), obj)

	// add 'a', verify present
	err = s.datastore.AddCollection(ctx, s.getTestCollection("a", nil))
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, "a")
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "a", obj.GetId())

	// dryrun duplicate 'a'
	err = s.datastore.DryRunCollection(ctx, s.getTestCollection("a", nil))
	assert.Error(s.T(), err)
	_, ok = err.(dag.VertexDuplicateError)
	assert.True(s.T(), ok)

	// try to add duplicate 'a'
	err = s.datastore.AddCollection(ctx, s.getTestCollection("a", nil))
	assert.Error(s.T(), err)
	_, ok = err.(dag.VertexDuplicateError)
	assert.True(s.T(), ok)

	// dryrun 'b' which points to 'a'
	err = s.datastore.DryRunCollection(ctx, s.getTestCollection("b", []string{"a"}))
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, "b")
	assert.NoError(s.T(), err)
	assert.False(s.T(), ok)
	assert.Nil(s.T(), obj)

	// add 'b' which points to 'a'
	err = s.datastore.AddCollection(ctx, s.getTestCollection("b", []string{"a"}))
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, "b")
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "b", obj.GetId())

	// try to delete 'a' while 'b' points to it
	err = s.datastore.DeleteCollection(ctx, "a")
	assert.Error(s.T(), err)

	// dryrun 'c' which has a self reference
	err = s.datastore.DryRunCollection(ctx, s.getTestCollection("c", []string{"c"}))
	assert.Error(s.T(), err)
	_, ok = err.(dag.SrcDstEqualError)
	assert.True(s.T(), ok)

	// try to add 'c' which has a self reference
	err = s.datastore.AddCollection(ctx, s.getTestCollection("c", []string{"c"}))
	assert.Error(s.T(), err)
	_, ok = err.(dag.SrcDstEqualError)
	assert.True(s.T(), ok)

	// dryrun 'd' which has a duplicate name
	obj = s.getTestCollection("c", nil)
	obj.Name = "a"
	err = s.datastore.DryRunCollection(ctx, obj)
	assert.Error(s.T(), err)
	_, ok = err.(dag.VertexDuplicateError)
	assert.True(s.T(), ok)

	// try to add 'd' which has duplicate name
	err = s.datastore.AddCollection(ctx, obj)
	assert.Error(s.T(), err)
	_, ok = err.(dag.VertexDuplicateError)
	assert.True(s.T(), ok)

	// clean up testing data
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, "b"))
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, "a"))
}

func (s *CollectionPostgresDataStoreTestSuite) TestFoo() {
	// TODO e2e testing ROX-12626
}

func (s *CollectionPostgresDataStoreTestSuite) getTestCollection(id string, ids []string) *storage.ResourceCollection {
	var embedded []*storage.ResourceCollection_EmbeddedResourceCollection
	if ids != nil {
		embedded = make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(ids))
		for _, i := range ids {
			embedded = append(embedded, &storage.ResourceCollection_EmbeddedResourceCollection{Id: i})
		}
	}
	return &storage.ResourceCollection{
		Id:                  id,
		Name:                id,
		EmbeddedCollections: embedded,
	}
}

func resetLocalGraph(ds *datastoreImpl) error {
	if ds.graph != nil {
		ds.graph = nil
	}
	return ds.initGraph()
}
