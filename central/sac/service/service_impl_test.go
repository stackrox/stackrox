package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/sac/datastore/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/secrets"
	"github.com/stretchr/testify/suite"
)

func TestSACService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(sacServiceTestSuite))
}

type sacServiceTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
	ds   *mocks.MockDataStore
	ctx  context.Context
	svc  *serviceImpl
}

func (s *sacServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.ds = mocks.NewMockDataStore(s.ctrl)
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	s.svc = &serviceImpl{s.ds}
}

func createAuthzPluginConfig() *storage.AuthzPluginConfig {
	return &storage.AuthzPluginConfig{
		Id:      "id",
		Name:    "name",
		Enabled: true,
		EndpointConfig: &storage.HTTPEndpointConfig{
			Endpoint:      "endpoint",
			Username:      "username",
			Password:      "password",
			ClientCertPem: "clientcertpem",
			ClientKeyPem:  "clientkeypem",
		},
	}
}

func (s *sacServiceTestSuite) TestGetAuthzPluginConfigs() {
	s.ds.EXPECT().ListAuthzPluginConfigs(gomock.Any()).Return(
		[]*storage.AuthzPluginConfig{
			createAuthzPluginConfig(),
			createAuthzPluginConfig(),
		}, nil)
	resp, err := s.svc.GetAuthzPluginConfigs(s.ctx, nil)
	s.NoError(err)
	s.Equal(len(resp.GetConfigs()), 2)
	for _, config := range resp.GetConfigs() {
		s.Equal(config.GetEndpointConfig().GetClientKeyPem(), secrets.ScrubReplacementStr)
		s.Equal(config.GetEndpointConfig().GetPassword(), secrets.ScrubReplacementStr)
		s.Equal(config.GetEndpointConfig().GetClientCertPem(), "clientcertpem")
	}
}
