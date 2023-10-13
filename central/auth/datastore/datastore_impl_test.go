//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/auth/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestAuthDatastorePostgres(t *testing.T) {
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	ctx       context.Context
	pool      *pgtest.TestPostgres
	store     pgStore.Store
	datastore DataStore
}

func (s *datastorePostgresTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access),
		),
	)

	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)

	s.store = pgStore.New(s.pool.DB)
	s.datastore = New(s.store)
}

func (s *datastorePostgresTestSuite) TearDownTest() {
	s.pool.Teardown(s.T())
	s.pool.Close()
}

func (s *datastorePostgresTestSuite) TestAddConfig() {
	testCases := []struct {
		config *storage.AuthMachineToMachineConfig
		err    error
	}{
		{
			config: &storage.AuthMachineToMachineConfig{},
			err:    errox.InvalidArgs,
		},
		{
			config: nil,
			err:    errox.InvalidArgs,
		},
		{
			config: &storage.AuthMachineToMachineConfig{
				Id: "some-id",
			},
			err: errox.InvalidArgs,
		},
		{
			config: &storage.AuthMachineToMachineConfig{
				TokenExpirationDuration: "1h",
			},
		},
		{
			config: &storage.AuthMachineToMachineConfig{
				TokenExpirationDuration: "1s",
			},
			err: errox.InvalidArgs,
		},
		{
			config: &storage.AuthMachineToMachineConfig{
				TokenExpirationDuration: "24h1s",
			},
			err: errox.InvalidArgs,
		},
		{
			config: &storage.AuthMachineToMachineConfig{
				TokenExpirationDuration: "1h",
				Type:                    storage.AuthMachineToMachineConfig_GENERIC,
			},
			err: errox.InvalidArgs,
		},
		{
			config: &storage.AuthMachineToMachineConfig{
				TokenExpirationDuration: "1h",
				Type:                    storage.AuthMachineToMachineConfig_GENERIC,
				IssuerConfig: &storage.AuthMachineToMachineConfig_Generic{
					Generic: &storage.AuthMachineToMachineConfig_GenericIssuer{Issuer: "something"}},
			},
		},
	}

	for i, tc := range testCases {
		s.Run(fmt.Sprintf("tc %d", i), func() {
			config, err := s.datastore.AddAuthM2MConfig(s.ctx, tc.config)
			if tc.err != nil {
				s.ErrorIs(err, tc.err)
			} else {
				s.NoError(err)
				s.NotEmpty(config.GetId())
			}
		})
	}
}

func (s *datastorePostgresTestSuite) TestGetConfig() {
	config, err := s.datastore.AddAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:   "sub",
				Value: "something",
				Role:  "Admin",
			},
			{
				Key:   "aud",
				Value: "github",
				Role:  "Continuous Integration",
			},
		},
	})
	s.Require().NoError(err)

	storedConfig, exists, err := s.datastore.GetAuthM2MConfig(s.ctx, config.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(config, storedConfig)
}

func (s *datastorePostgresTestSuite) TestListConfigs() {
	config1, err := s.datastore.AddAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:   "sub",
				Value: "something",
				Role:  "Admin",
			},
			{
				Key:   "aud",
				Value: "github",
				Role:  "Continuous Integration",
			},
		},
	})
	s.Require().NoError(err)

	config2, err := s.datastore.AddAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:   "sub",
				Value: "something",
				Role:  "Admin",
			},
			{
				Key:   "aud",
				Value: "github",
				Role:  "Continuous Integration",
			},
		},
	})
	s.Require().NoError(err)

	configs, err := s.datastore.ListAuthM2MConfigs(s.ctx)
	s.NoError(err)

	s.ElementsMatch(configs, []*storage.AuthMachineToMachineConfig{config1, config2})
}

func (s *datastorePostgresTestSuite) TestUpdateConfig() {
	config, err := s.datastore.AddAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:   "sub",
				Value: "something",
				Role:  "Admin",
			},
			{
				Key:   "aud",
				Value: "github",
				Role:  "Continuous Integration",
			},
		},
	})
	s.Require().NoError(err)

	config.Mappings = []*storage.AuthMachineToMachineConfig_Mapping{
		{
			Key:   "sub",
			Value: "someone",
			Role:  "SuperUser",
		},
	}

	updatedConfig, err := s.datastore.UpdateAuthM2MConfig(s.ctx, config)
	s.NoError(err)
	s.Equal(config, updatedConfig)
}

func (s *datastorePostgresTestSuite) TestRemoveConfig() {
	config, err := s.datastore.AddAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:   "sub",
				Value: "something",
				Role:  "Admin",
			},
			{
				Key:   "aud",
				Value: "github",
				Role:  "Continuous Integration",
			},
		},
	})
	s.Require().NoError(err)

	s.Error(s.datastore.RemoveAuthM2MConfig(s.ctx, "non-existing"))

	s.NoError(s.datastore.RemoveAuthM2MConfig(s.ctx, config.GetId()))

	config, exists, err := s.datastore.GetAuthM2MConfig(s.ctx, config.GetId())
	s.NoError(err)
	s.Empty(config)
	s.False(exists)
}
