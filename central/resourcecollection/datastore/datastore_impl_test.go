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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestCollectionDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(CollectionPostgresDataStoreTestSuite))
}

type CollectionPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        *pgxpool.Pool
	gormDB    *gorm.DB
	store     postgres.Store
	datastore DataStore
}

func (s *CollectionPostgresDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

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
			desc: "Test Graph Init graphInitBatchSize-1",
			size: graphInitBatchSize - 1,
		},
		{
			desc: "Test Graph Init graphInitBatchSize",
			size: graphInitBatchSize,
		},
		{
			desc: "Test Graph Init graphInitBatchSize+1",
			size: graphInitBatchSize + 1,
		},
		{
			desc: "Test Graph Init graphInitBatchSize+2",
			size: graphInitBatchSize + 2,
		},
		{
			desc: "Test Graph Init graphInitBatchSize*2",
			size: graphInitBatchSize * 2,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			objs := make([]*storage.ResourceCollection, 0, tc.size)
			objIDs := make([]string, 0, tc.size+1)

			obj := getTestCollection("0", nil)
			obj.Id = "0"
			objs = append(objs, obj)
			objIDs = append(objIDs, "0")
			for i := 1; i < tc.size; i++ {
				edges := make([]string, 0, i)
				for j := 0; j < i; j++ {
					edges = append(edges, fmt.Sprintf("%d", j))
				}
				id := fmt.Sprintf("%d", i)
				obj = getTestCollection(id, edges)
				obj.Id = id
				objs = append(objs, obj)
				objIDs = append(objIDs, id)
			}

			// add objs directly through the store
			err := s.store.UpsertMany(ctx, objs)
			assert.NoError(s.T(), err)

			// trigger graph init
			err = resetLocalGraph(s.datastore.(*datastoreImpl))
			assert.NoError(s.T(), err)

			// get data and check it
			batch, err := s.datastore.GetMany(ctx, objIDs)
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

	var err error

	// dryrun add object with an id set
	objID := getTestCollection("id", nil)
	objID.Id = "id"
	err = s.datastore.DryRunAddCollection(ctx, objID)
	assert.Error(s.T(), err)

	// add object with an id set
	err = s.datastore.AddCollection(ctx, objID)
	assert.Error(s.T(), err)

	// dryrun add 'a', verify not present
	objA := getTestCollection("a", nil)
	err = s.datastore.DryRunAddCollection(ctx, objA)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "", objA.Id)
	count, err := s.datastore.Count(ctx, nil)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, count)

	// add 'a', verify present
	err = s.datastore.AddCollection(ctx, objA)
	assert.NoError(s.T(), err)
	assert.NotEqual(s.T(), "", objA.Id)
	obj, ok, err := s.datastore.Get(ctx, objA.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objA, obj)

	// dryrun add duplicate 'a'
	objADup := getTestCollection("a", nil)
	err = s.datastore.DryRunAddCollection(ctx, objADup)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", objADup.Id)

	// add duplicate 'a'
	err = s.datastore.AddCollection(ctx, objADup)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", objADup.Id)

	// dryrun add 'b' which points to 'a', verify not present
	objB := getTestCollection("b", []string{objA.GetId()})
	err = s.datastore.DryRunAddCollection(ctx, objB)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "", objB.Id)
	count, err = s.datastore.Count(ctx, nil)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	// add 'b' which points to 'a', verify present
	err = s.datastore.AddCollection(ctx, objB)
	assert.NoError(s.T(), err)
	assert.NotEqual(s.T(), "", objB.Id)
	obj, ok, err = s.datastore.Get(ctx, objB.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objB, obj)

	// try to delete 'a' while 'b' points to it
	err = s.datastore.DeleteCollection(ctx, objA.GetId())
	assert.Error(s.T(), err)

	// try to update 'a' to point to 'b' which creates a cycle
	objACycle := getTestCollection("a", []string{objB.GetId()})
	objACycle.Id = objA.GetId()
	err = s.datastore.UpdateCollection(ctx, objACycle)
	assert.Error(s.T(), err)
	_, ok = err.(dag.EdgeLoopError)
	assert.True(s.T(), ok)

	// try to update 'a' to point to itself which creates a self cycle
	updateTestCollection(objACycle, []string{objA.GetId()})
	err = s.datastore.UpdateCollection(ctx, objACycle)
	assert.Error(s.T(), err)
	_, ok = err.(dag.SrcDstEqualError)
	assert.True(s.T(), ok)

	// try to update 'a' with a duplicate name
	objADup.Id = objA.GetId()
	objADup.Name = objB.GetName()
	err = s.datastore.UpdateCollection(ctx, objADup)
	assert.Error(s.T(), err)

	// try to update 'a' with a new name
	objA.Name = "A"
	err = s.datastore.UpdateCollection(ctx, objA)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objA.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objA, obj)

	// add 'e' that points to 'b' and verify
	objE := getTestCollection("e", []string{objB.GetId()})
	err = s.datastore.AddCollection(ctx, objE)
	assert.NoError(s.T(), err)
	assert.NotEqual(s.T(), "", objE.Id)
	obj, ok, err = s.datastore.Get(ctx, objE.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objE, obj)

	// update 'e' to point to only 'a', this tests addition and removal of edges
	updateTestCollection(objE, []string{objA.GetId()})
	err = s.datastore.UpdateCollection(ctx, objE)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objE.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objE, obj)

	// update 'b' to point to only 'e', making sure the original 'e' -> 'b' edge was removed
	updateTestCollection(objB, []string{objE.GetId()})
	err = s.datastore.UpdateCollection(ctx, objB)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objB.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objB, obj)

	// clean up testing data and verify the datastore is empty
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objB.GetId()))
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objE.GetId()))
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objA.GetId()))
	count, err = s.datastore.Count(ctx, nil)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, count)
}

func (s *CollectionPostgresDataStoreTestSuite) TestFoo() {
	// TODO e2e testing ROX-12626
}

func getTestCollection(name string, ids []string) *storage.ResourceCollection {
	return &storage.ResourceCollection{
		Name:                name,
		EmbeddedCollections: getEmbeddedTestCollection(ids),
	}
}

func updateTestCollection(obj *storage.ResourceCollection, ids []string) *storage.ResourceCollection {
	obj.EmbeddedCollections = getEmbeddedTestCollection(ids)
	return obj
}

func getEmbeddedTestCollection(ids []string) []*storage.ResourceCollection_EmbeddedResourceCollection {
	var embedded []*storage.ResourceCollection_EmbeddedResourceCollection
	if ids != nil {
		embedded = make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(ids))
		for _, i := range ids {
			embedded = append(embedded, &storage.ResourceCollection_EmbeddedResourceCollection{Id: i})
		}
	}
	return embedded
}

func resetLocalGraph(ds *datastoreImpl) error {
	if ds.graph != nil {
		ds.graph = nil
	}
	return ds.initGraph()
}
