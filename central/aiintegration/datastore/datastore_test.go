//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/aiintegration/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestAiIntegrationDatastorePostgres(t *testing.T) {
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	readCtx      context.Context
	writeCtx     context.Context
	noAccessCtx  context.Context
	postgresTest *pgtest.TestPostgres
	datastore    DataStore
}

func (s *datastorePostgresTestSuite) SetupTest() {
	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	store := pgStore.New(s.postgresTest.DB)
	s.datastore = New(store)
}

func (s *datastorePostgresTestSuite) TestUpsertAndGet() {
	integration := &storage.AiIntegration{
		Id:         uuid.NewV4().String(),
		Name:       "test-ols",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.openshift-lightspeed.svc:8443",
	}

	err := s.datastore.Upsert(s.writeCtx, integration)
	s.Require().NoError(err)

	got, exists, err := s.datastore.Get(s.readCtx, integration.GetId())
	s.Require().NoError(err)
	s.True(exists)
	protoassert.Equal(s.T(), integration, got)
}

func (s *datastorePostgresTestSuite) TestGetNonExistent() {
	got, exists, err := s.datastore.Get(s.readCtx, uuid.NewV4().String())
	s.NoError(err)
	s.False(exists)
	s.Nil(got)
}

func (s *datastorePostgresTestSuite) TestGetAll() {
	integrations, err := s.datastore.GetAll(s.readCtx)
	s.Require().NoError(err)
	s.Empty(integrations)

	for i := range 3 {
		err := s.datastore.Upsert(s.writeCtx, &storage.AiIntegration{
			Id:         uuid.NewV4().String(),
			Name:       fmt.Sprintf("ols-%d", i),
			Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
			ServiceUrl: fmt.Sprintf("https://ols-%d.example.com", i),
		})
		s.Require().NoError(err)
	}

	integrations, err = s.datastore.GetAll(s.readCtx)
	s.Require().NoError(err)
	s.Len(integrations, 3)
}

func (s *datastorePostgresTestSuite) TestUpsertOverwrite() {
	id := uuid.NewV4().String()

	err := s.datastore.Upsert(s.writeCtx, &storage.AiIntegration{
		Id:         id,
		Name:       "original",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://original.example.com",
	})
	s.Require().NoError(err)

	err = s.datastore.Upsert(s.writeCtx, &storage.AiIntegration{
		Id:         id,
		Name:       "updated",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://updated.example.com",
	})
	s.Require().NoError(err)

	got, exists, err := s.datastore.Get(s.readCtx, id)
	s.Require().NoError(err)
	s.True(exists)
	s.Equal("updated", got.GetName())
	s.Equal("https://updated.example.com", got.GetServiceUrl())
}

func (s *datastorePostgresTestSuite) TestDelete() {
	id := uuid.NewV4().String()
	err := s.datastore.Upsert(s.writeCtx, &storage.AiIntegration{
		Id:         id,
		Name:       "to-delete",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com",
	})
	s.Require().NoError(err)

	err = s.datastore.Delete(s.writeCtx, id)
	s.Require().NoError(err)

	_, exists, err := s.datastore.Get(s.readCtx, id)
	s.NoError(err)
	s.False(exists)
}

func (s *datastorePostgresTestSuite) TestDeleteNonExistent() {
	err := s.datastore.Delete(s.writeCtx, uuid.NewV4().String())
	s.NoError(err)
}

func (s *datastorePostgresTestSuite) TestExists() {
	id := uuid.NewV4().String()

	exists, err := s.datastore.Exists(s.readCtx, id)
	s.NoError(err)
	s.False(exists)

	err = s.datastore.Upsert(s.writeCtx, &storage.AiIntegration{
		Id:         id,
		Name:       "exists-test",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com",
	})
	s.Require().NoError(err)

	exists, err = s.datastore.Exists(s.readCtx, id)
	s.NoError(err)
	s.True(exists)
}

func (s *datastorePostgresTestSuite) TestSACReadDenied() {
	id := uuid.NewV4().String()
	err := s.datastore.Upsert(s.writeCtx, &storage.AiIntegration{
		Id:         id,
		Name:       "sac-test",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com",
	})
	s.Require().NoError(err)

	got, exists, err := s.datastore.Get(s.noAccessCtx, id)
	s.NoError(err)
	s.False(exists)
	s.Nil(got)

	integrations, err := s.datastore.GetAll(s.noAccessCtx)
	s.NoError(err)
	s.Nil(integrations)

	existsResult, err := s.datastore.Exists(s.noAccessCtx, id)
	s.NoError(err)
	s.False(existsResult)
}

func (s *datastorePostgresTestSuite) TestSACWriteDenied() {
	err := s.datastore.Upsert(s.noAccessCtx, &storage.AiIntegration{
		Id:         uuid.NewV4().String(),
		Name:       "denied",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com",
	})
	s.Error(err)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	err = s.datastore.Delete(s.noAccessCtx, uuid.NewV4().String())
	s.Error(err)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}
