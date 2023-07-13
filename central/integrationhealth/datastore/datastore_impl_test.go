//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	postgresIntegrationStore "github.com/stackrox/rox/central/integrationhealth/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationHealthDatastore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(integrationHealthDatastoreTestSuite))
}

type integrationHealthDatastoreTestSuite struct {
	suite.Suite

	hasReadCtx     context.Context
	hasWriteCtx    context.Context
	hasNoAccessCtx context.Context

	datastore    DataStore
	postgresTest *pgtest.TestPostgres
}

func (s *integrationHealthDatastoreTestSuite) SetupTest() {
	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	integrationStore := postgresIntegrationStore.New(s.postgresTest.DB)
	s.datastore = New(integrationStore)

	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)

	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.hasNoAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
}

func (s *integrationHealthDatastoreTestSuite) TearDownTest() {
	s.postgresTest.Close()
}

func (s *integrationHealthDatastoreTestSuite) TestGetRegistriesAndScanners() {
	integrationHealth := newIntegrationHealth(storage.IntegrationHealth_IMAGE_INTEGRATION)

	err := s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.NoError(err)

	s.testGetIntegrationHealth(s.datastore.GetRegistriesAndScanners)

	receivedIntegrationHealths, err := s.datastore.GetRegistriesAndScanners(s.hasReadCtx)
	s.NoError(err)
	s.ElementsMatch([]*storage.IntegrationHealth{integrationHealth}, receivedIntegrationHealths)
}

func (s *integrationHealthDatastoreTestSuite) TestGetNotifierPlugins() {
	integrationHealth := newIntegrationHealth(storage.IntegrationHealth_NOTIFIER)

	err := s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.NoError(err)

	s.testGetIntegrationHealth(s.datastore.GetNotifierPlugins)

	receivedIntegrationHealths, err := s.datastore.GetNotifierPlugins(s.hasReadCtx)
	s.NoError(err)
	s.ElementsMatch([]*storage.IntegrationHealth{integrationHealth}, receivedIntegrationHealths)
}

func (s *integrationHealthDatastoreTestSuite) TestGetBackupPlugins() {
	integrationHealth := newIntegrationHealth(storage.IntegrationHealth_BACKUP)

	err := s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.NoError(err)

	s.testGetIntegrationHealth(s.datastore.GetBackupPlugins)

	receivedIntegrationHealths, err := s.datastore.GetBackupPlugins(s.hasReadCtx)
	s.NoError(err)
	s.ElementsMatch([]*storage.IntegrationHealth{integrationHealth}, receivedIntegrationHealths)
}

func (s *integrationHealthDatastoreTestSuite) TestGetDeclarativeConfigs() {
	integrationHealth := newIntegrationHealth(storage.IntegrationHealth_DECLARATIVE_CONFIG)

	err := s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.NoError(err)

	s.testGetIntegrationHealth(s.datastore.GetDeclarativeConfigs)

	receivedIntegrationHealths, err := s.datastore.GetDeclarativeConfigs(s.hasReadCtx)
	s.NoError(err)
	s.ElementsMatch([]*storage.IntegrationHealth{integrationHealth}, receivedIntegrationHealths)
}

func (s *integrationHealthDatastoreTestSuite) TestUpdateIntegrationHealth() {
	integrationHealth := newIntegrationHealth(storage.IntegrationHealth_IMAGE_INTEGRATION)

	// 1. With no access should return no error but should not be added.
	err := s.datastore.UpsertIntegrationHealth(s.hasNoAccessCtx, integrationHealth)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err := s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 2. With READ access should return no error but should not be added.
	err = s.datastore.UpsertIntegrationHealth(s.hasReadCtx, integrationHealth)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err = s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 3. With WRITE access should not return an error and the integration should be retrievable.
	err = s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.NoError(err)
	receivedIntegrationHealth, exists, err := s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(integrationHealth, receivedIntegrationHealth)

	// 4. Updating an invalid integration health type should not be possible.
	integrationHealth.Type = storage.IntegrationHealth_UNKNOWN
	err = s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *integrationHealthDatastoreTestSuite) TestRemoveIntegrationHealth() {
	integrationHealth := newIntegrationHealth(storage.IntegrationHealth_IMAGE_INTEGRATION)
	err := s.datastore.UpsertIntegrationHealth(s.hasWriteCtx, integrationHealth)
	s.NoError(err)
	_, exists, err := s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 1. With no access should not return an error but integration should still exist.
	err = s.datastore.RemoveIntegrationHealth(s.hasNoAccessCtx, integrationHealth.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err = s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 2. With READ access should not return an error but integration should still exist.
	err = s.datastore.RemoveIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	_, exists, err = s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.True(exists)

	// 3. With WRITE access and existing integration health remove should not return an error.

	err = s.datastore.RemoveIntegrationHealth(s.hasWriteCtx, integrationHealth.GetId())
	s.NoError(err)
	_, exists, err = s.datastore.GetIntegrationHealth(s.hasReadCtx, integrationHealth.GetId())
	s.NoError(err)
	s.False(exists)

	// 4. With WRITE access should return an error if integration health is not found.
	err = s.datastore.RemoveIntegrationHealth(s.hasWriteCtx, integrationHealth.GetId())
	s.Error(err, errox.NotFound)
}

func (s *integrationHealthDatastoreTestSuite) testGetIntegrationHealth(getIntegrationHealth func(ctx context.Context) ([]*storage.IntegrationHealth, error)) {
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

func newIntegrationHealth(typ storage.IntegrationHealth_Type) *storage.IntegrationHealth {
	return &storage.IntegrationHealth{
		Id:           uuid.NewV4().String(),
		Name:         "",
		Type:         typ,
		Status:       0,
		ErrorMessage: "",
	}
}
