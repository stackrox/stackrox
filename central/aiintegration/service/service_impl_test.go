//go:build sql_integration

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/central/aiintegration/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAiIntegrationService(t *testing.T) {
	suite.Run(t, new(aiIntegrationServiceTestSuite))
}

type aiIntegrationServiceTestSuite struct {
	suite.Suite

	ctx  context.Context
	pool *pgtest.TestPostgres
	ds   datastore.DataStore
	svc  Service
}

func (s *aiIntegrationServiceTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)
	s.ds = datastore.GetTestPostgresDataStore(s.T(), s.pool.DB)
	s.svc = New(s.ds)
}

func (s *aiIntegrationServiceTestSuite) TestCreateAiIntegration() {
	req := &v2.AiIntegration{
		Name:       "test-ols",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.openshift-lightspeed.svc:8443",
	}

	resp, err := s.svc.CreateAiIntegration(s.ctx, req)
	s.Require().NoError(err)
	s.NotEmpty(resp.GetId())
	s.Equal("test-ols", resp.GetName())
	s.Equal(v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS, resp.GetType())
	s.Equal("https://ols.openshift-lightspeed.svc:8443", resp.GetServiceUrl())

	got, exists, err := s.ds.Get(s.ctx, resp.GetId())
	s.Require().NoError(err)
	s.True(exists)
	s.Equal("test-ols", got.GetName())
}

func (s *aiIntegrationServiceTestSuite) TestCreateAiIntegration_ValidationErrors() {
	tests := map[string]struct {
		req     *v2.AiIntegration
		wantMsg string
	}{
		"nil request": {
			req:     nil,
			wantMsg: "integration must be provided",
		},
		"empty name": {
			req: &v2.AiIntegration{
				Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
				ServiceUrl: "https://example.com",
			},
			wantMsg: "integration name must be provided",
		},
		"unspecified type": {
			req: &v2.AiIntegration{
				Name:       "test",
				Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_UNSPECIFIED,
				ServiceUrl: "https://example.com",
			},
			wantMsg: "integration type must be specified",
		},
		"empty service_url": {
			req: &v2.AiIntegration{
				Name: "test",
				Type: v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
			},
			wantMsg: "service_url must be provided",
		},
		"invalid service_url": {
			req: &v2.AiIntegration{
				Name:       "test",
				Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
				ServiceUrl: "not-a-url",
			},
			wantMsg: "invalid service_url",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			resp, err := s.svc.CreateAiIntegration(s.ctx, tc.req)
			s.Nil(resp)
			s.Require().Error(err)
			s.ErrorIs(err, errox.InvalidArgs)
			s.Contains(err.Error(), tc.wantMsg)
		})
	}
}

func (s *aiIntegrationServiceTestSuite) TestGetAiIntegration() {
	id := uuid.NewV4().String()
	s.Require().NoError(s.ds.Upsert(s.ctx, &storage.AiIntegration{
		Id:         id,
		Name:       "my-ols",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com",
	}))

	resp, err := s.svc.GetAiIntegration(s.ctx, &v2.ResourceByID{Id: id})
	s.Require().NoError(err)
	s.Equal(id, resp.GetId())
	s.Equal("my-ols", resp.GetName())
	s.Equal(v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS, resp.GetType())
	s.Equal("https://ols.example.com", resp.GetServiceUrl())
}

func (s *aiIntegrationServiceTestSuite) TestGetAiIntegration_NotFound() {
	resp, err := s.svc.GetAiIntegration(s.ctx, &v2.ResourceByID{Id: uuid.NewV4().String()})
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.NotFound)
}

func (s *aiIntegrationServiceTestSuite) TestGetAiIntegration_EmptyID() {
	resp, err := s.svc.GetAiIntegration(s.ctx, &v2.ResourceByID{Id: ""})
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *aiIntegrationServiceTestSuite) TestListAiIntegrations() {
	for i := range 3 {
		s.Require().NoError(s.ds.Upsert(s.ctx, &storage.AiIntegration{
			Id:         uuid.NewV4().String(),
			Name:       "ols-" + uuid.NewV4().String()[:8],
			Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
			ServiceUrl: "https://ols.example.com",
		}))
		_ = i
	}

	resp, err := s.svc.ListAiIntegrations(s.ctx, &v2.Empty{})
	s.Require().NoError(err)
	s.Len(resp.GetIntegrations(), 3)
}

func (s *aiIntegrationServiceTestSuite) TestListAiIntegrations_Empty() {
	resp, err := s.svc.ListAiIntegrations(s.ctx, &v2.Empty{})
	s.Require().NoError(err)
	s.Empty(resp.GetIntegrations())
}

func (s *aiIntegrationServiceTestSuite) TestUpdateAiIntegration() {
	id := uuid.NewV4().String()
	s.Require().NoError(s.ds.Upsert(s.ctx, &storage.AiIntegration{
		Id:         id,
		Name:       "old-name",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://old.example.com",
	}))

	req := &v2.AiIntegration{
		Id:         id,
		Name:       "new-name",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://new.example.com",
	}

	resp, err := s.svc.UpdateAiIntegration(s.ctx, req)
	s.Require().NoError(err)
	s.NotNil(resp)

	got, exists, err := s.ds.Get(s.ctx, id)
	s.Require().NoError(err)
	s.True(exists)
	s.Equal("new-name", got.GetName())
	s.Equal("https://new.example.com", got.GetServiceUrl())
}

func (s *aiIntegrationServiceTestSuite) TestUpdateAiIntegration_NotFound() {
	req := &v2.AiIntegration{
		Id:         uuid.NewV4().String(),
		Name:       "test",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://example.com",
	}

	resp, err := s.svc.UpdateAiIntegration(s.ctx, req)
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.NotFound)
}

func (s *aiIntegrationServiceTestSuite) TestUpdateAiIntegration_EmptyID() {
	req := &v2.AiIntegration{
		Name:       "test",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://example.com",
	}

	resp, err := s.svc.UpdateAiIntegration(s.ctx, req)
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *aiIntegrationServiceTestSuite) TestDeleteAiIntegration() {
	id := uuid.NewV4().String()
	s.Require().NoError(s.ds.Upsert(s.ctx, &storage.AiIntegration{
		Id:         id,
		Name:       "to-delete",
		Type:       storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com",
	}))

	resp, err := s.svc.DeleteAiIntegration(s.ctx, &v2.ResourceByID{Id: id})
	s.Require().NoError(err)
	s.NotNil(resp)

	_, exists, err := s.ds.Get(s.ctx, id)
	s.Require().NoError(err)
	s.False(exists)
}

func (s *aiIntegrationServiceTestSuite) TestDeleteAiIntegration_NotFound() {
	resp, err := s.svc.DeleteAiIntegration(s.ctx, &v2.ResourceByID{Id: uuid.NewV4().String()})
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.NotFound)
}

func (s *aiIntegrationServiceTestSuite) TestDeleteAiIntegration_EmptyID() {
	resp, err := s.svc.DeleteAiIntegration(s.ctx, &v2.ResourceByID{Id: ""})
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *aiIntegrationServiceTestSuite) TestTestAiIntegration_Success() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/readiness" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	req := &v2.AiIntegration{
		Name:       "test-ols",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: server.URL,
	}

	resp, err := s.svc.TestAiIntegration(s.ctx, req)
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *aiIntegrationServiceTestSuite) TestTestAiIntegration_Unavailable() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	req := &v2.AiIntegration{
		Name:       "test-ols",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: server.URL,
	}

	resp, err := s.svc.TestAiIntegration(s.ctx, req)
	s.Nil(resp)
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.True(ok)
	s.Equal(codes.Unavailable, st.Code())
}

func (s *aiIntegrationServiceTestSuite) TestTestAiIntegration_ConnectionRefused() {
	req := &v2.AiIntegration{
		Name:       "test-ols",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "http://127.0.0.1:1",
	}

	resp, err := s.svc.TestAiIntegration(s.ctx, req)
	s.Nil(resp)
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.True(ok)
	s.Equal(codes.Unavailable, st.Code())
}

func (s *aiIntegrationServiceTestSuite) TestTestAiIntegration_ValidationError() {
	req := &v2.AiIntegration{
		Name: "test-ols",
		Type: v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
	}

	resp, err := s.svc.TestAiIntegration(s.ctx, req)
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func TestApiToStorageConversion(t *testing.T) {
	api := &v2.AiIntegration{
		Id:         "abc-123",
		Name:       "my-integration",
		Type:       v2.AiIntegrationType_AI_INTEGRATION_TYPE_OLS,
		ServiceUrl: "https://ols.example.com:8443",
	}

	s := apiToStorage(api)
	assert.Equal(t, "abc-123", s.GetId())
	assert.Equal(t, "my-integration", s.GetName())
	assert.Equal(t, storage.AiIntegrationType_AI_INTEGRATION_TYPE_OLS, s.GetType())
	assert.Equal(t, "https://ols.example.com:8443", s.GetServiceUrl())

	back := storageToAPI(s)
	assert.Equal(t, api.GetId(), back.GetId())
	assert.Equal(t, api.GetName(), back.GetName())
	assert.Equal(t, api.GetType(), back.GetType())
	assert.Equal(t, api.GetServiceUrl(), back.GetServiceUrl())
}
