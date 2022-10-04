//go:build sql_integration

package datastore

import (
	"context"
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
	"github.com/stretchr/testify/suite"
)

func TestCollectionDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(CollectionPostgresDataStoreTestSuite))
}

type CollectionPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx         context.Context
	db          *pgxpool.Pool
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

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := postgres.CreateTableAndNewStore(s.ctx, s.db, gormDB)
	index := postgres.NewIndexer(s.db)
	s.datastore = New(store, index, search.New(store, index))
}

func (s *CollectionPostgresDataStoreTestSuite) TearDownSuite() {
	postgres.Destroy(s.ctx, s.db)
	s.db.Close()
	s.envIsolator.RestoreAll()
}

func (s *CollectionPostgresDataStoreTestSuite) TestAddCollection() {
	ctx := sac.WithAllAccess(context.Background())

	// add 'a'
	err := s.datastore.AddCollection(ctx, s.getTestCollection("a", nil))
	s.NoError(err)
	obj, ok, err := s.datastore.Get(ctx, "a")
	s.NoError(err)
	s.True(ok)
	s.Equal("a", obj.GetId())

	// try to add duplicate 'a' and check that we fail
	err = s.datastore.AddCollection(ctx, s.getTestCollection("a", nil))
	s.NotNil(err)
	_, ok = err.(dag.VertexDuplicateError)
	s.True(ok)

	// add 'b' which points to 'a'
	err = s.datastore.AddCollection(ctx, s.getTestCollection("b", []string{"a"}))
	s.NoError(err)
	obj, ok, err = s.datastore.Get(ctx, "b")
	s.NoError(err)
	s.True(ok)
	s.Equal("b", obj.GetId())

	// try to delete 'a' while 'b' points to it
	err = s.datastore.DeleteCollection(ctx, "a")
	s.NotNil(err)

	// try to add 'c' which has a self reference
	err = s.datastore.AddCollection(ctx, s.getTestCollection("c", []string{"c"}))
	s.NotNil(err)
	_, ok = err.(dag.SrcDstEqualError)
	s.True(ok)
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
