//go:build sql_integration

package health

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/declarativeconfig/health/store"
	postgresHealthStore "github.com/stackrox/rox/central/declarativeconfig/health/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestDeclarativeConfigHealthDatastore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(declarativeConfigHealthDatastoreSuite))
}

type declarativeConfigHealthDatastoreSuite struct {
	suite.Suite

	hasReadCtx     context.Context
	hasWriteCtx    context.Context
	hasNoAccessCtx context.Context

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
	s.hasNoAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
}

func (s *declarativeConfigHealthDatastoreSuite) TearDownTest() {
	s.postgresTest.Close()
}

func (s *declarativeConfigHealthDatastoreSuite) TestGetDeclarativeConfigs() {
	integrationHealth := newIntegrationHealth()

	err := s.datastore.UpsertDeclarativeConfig(s.hasWriteCtx, integrationHealth)
	s.NoError(err)

	s.testGetIntegrationHealth(s.datastore.GetDeclarativeConfigs)

	receivedIntegrationHealths, err := s.datastore.GetDeclarativeConfigs(s.hasReadCtx)
	s.NoError(err)
	s.ElementsMatch([]*storage.DeclarativeConfigHealth{integrationHealth}, receivedIntegrationHealths)
}

func (s *declarativeConfigHealthDatastoreSuite) TestUpdateIntegrationHealth() {
	integrationHealth := newIntegrationHealth()

	// 1. With no access should return no error but should not be added.
	err := s.datastore.UpsertDeclarativeConfig(s.hasNoAccessCtx, integrationHealth)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err := s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 2. With READ access should return no error but should not be added.
	err = s.datastore.UpsertDeclarativeConfig(s.hasReadCtx, integrationHealth)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 3. With WRITE access should not return an error and the integration should be retrievable.
	err = s.datastore.UpsertDeclarativeConfig(s.hasWriteCtx, integrationHealth)
	s.NoError(err)
	receivedIntegrationHealth, exists, err := s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(integrationHealth, receivedIntegrationHealth)
}

func (s *declarativeConfigHealthDatastoreSuite) TestRemoveIntegrationHealth() {
	integrationHealth := newIntegrationHealth()
	err := s.datastore.UpsertDeclarativeConfig(s.hasWriteCtx, integrationHealth)
	s.NoError(err)
	_, exists, err := s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 1. With no access should not return an error but integration should still exist.
	err = s.datastore.RemoveDeclarativeConfig(s.hasNoAccessCtx, integrationHealth.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 2. With READ access should not return an error but integration should still exist.
	err = s.datastore.RemoveDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 3. With WRITE access and existing integration health remove should not return an error.

	err = s.datastore.RemoveDeclarativeConfig(s.hasWriteCtx, integrationHealth.GetId())
	s.NoError(err)
	_, exists, err = s.datastore.GetDeclarativeConfig(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 4. With WRITE access should return an error if integration health is not found.
	err = s.datastore.RemoveDeclarativeConfig(s.hasWriteCtx, integrationHealth.GetId())
	s.Error(err, errox.NotFound)
}

func (s *declarativeConfigHealthDatastoreSuite) testGetIntegrationHealth(getIntegrationHealth func(ctx context.Context) ([]*storage.DeclarativeConfigHealth, error)) {
	// 1. With no access should not return an error and no integration.
	integration, err := getIntegrationHealth(s.hasNoAccessCtx)
	s.NoError(err)
	s.Nil(integration)

	// 2. With READ access should return an integration.
	integration, err = getIntegrationHealth(s.hasReadCtx)
	s.NoError(err)
	s.NotNil(integration)

	// 3. With WRITE access should return an integration.
	integration, err = getIntegrationHealth(s.hasWriteCtx)
	s.NoError(err)
	s.NotNil(integration)
}

func newIntegrationHealth() *storage.DeclarativeConfigHealth {
	return &storage.DeclarativeConfigHealth{
		Id:           uuid.NewV4().String(),
		Name:         "",
		Status:       0,
		ErrorMessage: "",
	}
}
