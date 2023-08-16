//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/declarativeconfig/health/datastore/store"
	postgresHealthStore "github.com/stackrox/rox/central/declarativeconfig/health/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestDeclarativeConfigHealthDatastore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(declarativeConfigHealthDatastoreSuite))
}

type declarativeConfigHealthDatastoreSuite struct {
	suite.Suite

	hasReadCtx             context.Context
	hasWriteCtx            context.Context
	hasWriteDeclarativeCtx context.Context
	hasNoAccessCtx         context.Context

	datastore    DataStore
	postgresTest *pgtest.TestPostgres
}

func (s *declarativeConfigHealthDatastoreSuite) SetupTest() {
	var declarativeConfigHealthStore store.Store

	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	declarativeConfigHealthStore = postgresHealthStore.New(s.postgresTest.DB)
	s.datastore = New(declarativeConfigHealthStore)

	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)

	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.hasWriteDeclarativeCtx = declarativeconfig.WithModifyDeclarativeResource(s.hasWriteCtx)
	s.hasNoAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
}

func (s *declarativeConfigHealthDatastoreSuite) TearDownTest() {
	s.postgresTest.Close()
}

func (s *declarativeConfigHealthDatastoreSuite) TestGetDeclarativeConfigs() {
	configHealth := newConfigHealth()

	err := s.datastore.UpsertDeclarativeConfig(s.hasWriteDeclarativeCtx, configHealth)
	s.NoError(err)

	s.testGetConfigHealth(s.datastore.GetDeclarativeConfigs)

	receivedConfigHealths, err := s.datastore.GetDeclarativeConfigs(s.hasReadCtx)
	s.NoError(err)
	s.ElementsMatch([]*storage.DeclarativeConfigHealth{configHealth}, receivedConfigHealths)
}

func (s *declarativeConfigHealthDatastoreSuite) TestUpdateConfigHealth() {
	configHealth := newConfigHealth()

	// 1. With no access should return  error + should not be added.
	err := s.datastore.UpsertDeclarativeConfig(s.hasNoAccessCtx, configHealth)
	s.ErrorIs(err, errox.NotAuthorized)
	_, exists, err := s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 2. With READ access should return error + should not be added.
	err = s.datastore.UpsertDeclarativeConfig(s.hasReadCtx, configHealth)
	s.ErrorIs(err, errox.NotAuthorized)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 3. With WRITE access should return an error + should not be added.
	err = s.datastore.UpsertDeclarativeConfig(s.hasWriteCtx, configHealth)
	s.ErrorIs(err, errox.NotAuthorized)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 4. With WRITE access and declarative config context key should not return an error and the config
	// health should be retrievable.
	err = s.datastore.UpsertDeclarativeConfig(s.hasWriteDeclarativeCtx, configHealth)
	s.NoError(err)
	receivedConfigHealth, exists, err := s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(configHealth, receivedConfigHealth)
}

func (s *declarativeConfigHealthDatastoreSuite) TestRemoveConfigHealth() {
	configHealth := newConfigHealth()
	err := s.datastore.UpsertDeclarativeConfig(s.hasWriteDeclarativeCtx, configHealth)
	s.NoError(err)
	_, exists, err := s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 1. With no access should return an error + config health should still exist.
	err = s.datastore.RemoveDeclarativeConfig(s.hasNoAccessCtx, configHealth.GetId())
	s.ErrorIs(err, errox.NotAuthorized)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 2. With READ access should return an error + config health should still exist.
	err = s.datastore.RemoveDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.ErrorIs(err, errox.NotAuthorized)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 3. With WRITE access and existing config health remove should return an error.
	err = s.datastore.RemoveDeclarativeConfig(s.hasWriteCtx, configHealth.GetId())
	s.ErrorIs(err, errox.NotAuthorized)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 3. With WRITE access, declarative config context key, and existing config health remove should
	// not return an error.
	err = s.datastore.RemoveDeclarativeConfig(s.hasWriteDeclarativeCtx, configHealth.GetId())
	s.NoError(err)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, configHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 4. With WRITE access and declarative config context key should return an error if config health is not found.
	err = s.datastore.RemoveDeclarativeConfig(s.hasWriteDeclarativeCtx, configHealth.GetId())
	s.ErrorIs(err, errox.NotFound)
}

func (s *declarativeConfigHealthDatastoreSuite) testGetConfigHealth(getConfigHealth func(ctx context.Context) ([]*storage.DeclarativeConfigHealth, error)) {
	// 1. With no access should not return an error and no config health.
	configHealth, err := getConfigHealth(s.hasNoAccessCtx)
	s.NoError(err)
	s.Nil(configHealth)

	// 2. With READ access should return a config health
	configHealth, err = getConfigHealth(s.hasReadCtx)
	s.NoError(err)
	s.NotNil(configHealth)

	// 3. With WRITE access should return a config health.
	configHealth, err = getConfigHealth(s.hasWriteCtx)
	s.NoError(err)
	s.NotNil(configHealth)

	// 4. With WRITE access and declarative config context key should return a config health.
	configHealth, err = getConfigHealth(s.hasWriteDeclarativeCtx)
	s.NoError(err)
	s.NotNil(configHealth)
}

func newConfigHealth() *storage.DeclarativeConfigHealth {
	return &storage.DeclarativeConfigHealth{
		Id:           uuid.NewV4().String(),
		Name:         "",
		Status:       0,
		ErrorMessage: "",
	}
}
