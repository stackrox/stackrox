//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/central/auth/store"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	permissionSetPostgresStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	rolePostgresStore "github.com/stackrox/rox/central/role/store/role/postgres"
	accessScopePostgresStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testRole1        = "New-Admin"
	testRole2        = "Super-Admin"
	testRole3        = "Super Continuous Integration"
	configController = "Configuration Controller"
	testIssuer       = "https://localhost"
)

var (
	testRoles = set.NewFrozenStringSet(testRole1, testRole2, testRole3, configController)
)

func TestAuthDatastorePostgres(t *testing.T) {
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	ctx           context.Context
	pool          *pgtest.TestPostgres
	authDataStore DataStore
	roleDataStore roleDataStore.DataStore
	mockSet       *mocks.MockTokenExchangerSet
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

	store := store.New(s.pool.DB)

	permSetStore := permissionSetPostgresStore.New(s.pool.DB)
	accessScopeStore := accessScopePostgresStore.New(s.pool.DB)
	roleStore := rolePostgresStore.New(s.pool.DB)
	s.roleDataStore = roleDataStore.New(roleStore, permSetStore, accessScopeStore, func(_ context.Context, _ func(*storage.Group) bool) ([]*storage.Group, error) {
		return nil, nil
	})

	s.addRoles(roleStore)

	controller := gomock.NewController(s.T())
	s.mockSet = mocks.NewMockTokenExchangerSet(controller)
	s.mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.mockSet.EXPECT().RemoveTokenExchanger(gomock.Any()).Return(nil).AnyTimes()
	s.mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, true).AnyTimes()
	s.mockSet.EXPECT().RollbackExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	issuerFetcher := mocks.NewMockServiceAccountIssuerFetcher(controller)
	issuerFetcher.EXPECT().GetServiceAccountIssuer().Return("https://localhost", nil).AnyTimes()

	s.authDataStore = New(store, s.mockSet, issuerFetcher)
}

func (s *datastorePostgresTestSuite) TestKubeServiceAccountConfig() {
	controller := gomock.NewController(s.T())
	defer controller.Finish()
	store := store.New(s.pool.DB)

	mockSet := mocks.NewMockTokenExchangerSet(controller)
	issuerFetcher := mocks.NewMockServiceAccountIssuerFetcher(controller)

	issuerFetcher.EXPECT().GetServiceAccountIssuer().Return(testIssuer, nil).Times(1)
	mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, false).Times(1)

	authDataStore := New(store, mockSet, issuerFetcher)
	s.NoError(authDataStore.InitializeTokenExchangers())
}

func getTestConfig(configs []*storage.AuthMachineToMachineConfig) *storage.AuthMachineToMachineConfig {
	for _, config := range configs {
		if config.Issuer == testIssuer {
			return config
		}
	}
	return nil
}

type authDataStoreMutatorFunc func(authDataStore DataStore)
type authDataStoreValidatorFunc func(configs []*storage.AuthMachineToMachineConfig)

func (s *datastorePostgresTestSuite) kubeSAM2MConfig(authDataStoreMutator authDataStoreMutatorFunc, authDataStoreValidator authDataStoreValidatorFunc) {
	controller := gomock.NewController(s.T())
	defer controller.Finish()
	store := store.New(s.pool.DB)

	mockSet := mocks.NewMockTokenExchangerSet(controller)
	issuerFetcher := mocks.NewMockServiceAccountIssuerFetcher(controller)

	issuerFetcher.EXPECT().GetServiceAccountIssuer().Return(testIssuer, nil).Times(2)
	mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, false).AnyTimes()
	mockSet.EXPECT().RemoveTokenExchanger(gomock.AssignableToTypeOf("")).Return(nil).AnyTimes()

	authDataStore := New(store, mockSet, issuerFetcher)
	s.NoError(authDataStore.InitializeTokenExchangers())
	authDataStoreMutator(authDataStore)

	// Emulate restarting Central by creating a new data store and token exchanger set
	mockSet = mocks.NewMockTokenExchangerSet(controller)
	mockSet.EXPECT().UpsertTokenExchanger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSet.EXPECT().GetTokenExchanger(gomock.Any()).Return(nil, false).AnyTimes()

	authDataStore = New(store, mockSet, issuerFetcher)
	s.NoError(authDataStore.InitializeTokenExchangers())

	var configs []*storage.AuthMachineToMachineConfig
	err := authDataStore.ForEachAuthM2MConfig(s.ctx, func(obj *storage.AuthMachineToMachineConfig) error {
		configs = append(configs, obj)
		return nil
	})
	s.NoError(err)
	authDataStoreValidator(configs)
}

func (s *datastorePostgresTestSuite) TestKubeSAM2MConfigPersistsAfterDelete() {
	authDataStoreMutator := func(authDataStore DataStore) {
		var configs []*storage.AuthMachineToMachineConfig
		err := authDataStore.ForEachAuthM2MConfig(s.ctx, func(obj *storage.AuthMachineToMachineConfig) error {
			configs = append(configs, obj)
			return nil
		})
		s.NoError(err)
		s.NotEmpty(configs)
		kubeSAConfig := getTestConfig(configs)
		s.NotNil(kubeSAConfig)
		s.NoError(authDataStore.RemoveAuthM2MConfig(s.ctx, kubeSAConfig.Id))
	}
	authDataStoreValidator := func(configs []*storage.AuthMachineToMachineConfig) {
		kubeSAConfig := getTestConfig(configs)
		s.NotNil(kubeSAConfig)
		s.Equal(1, len(kubeSAConfig.Mappings))
		s.Equal("sub", kubeSAConfig.Mappings[0].Key)
		s.Equal("Configuration Controller", kubeSAConfig.Mappings[0].Role)
		s.Contains(kubeSAConfig.Mappings[0].ValueExpression, "config-controller")
	}

	s.kubeSAM2MConfig(authDataStoreMutator, authDataStoreValidator)
}

func (s *datastorePostgresTestSuite) TestKubeSAM2MConfigPersistsAfterRestart() {
	authDataStoreMutator := func(authDataStore DataStore) {}
	authDataStoreValidator := func(configs []*storage.AuthMachineToMachineConfig) {
		kubeSAConfig := getTestConfig(configs)
		s.NotNil(kubeSAConfig)
		s.Equal(1, len(kubeSAConfig.Mappings))
		s.Equal("sub", kubeSAConfig.Mappings[0].Key)
		s.Equal("Configuration Controller", kubeSAConfig.Mappings[0].Role)
		s.Contains(kubeSAConfig.Mappings[0].ValueExpression, "config-controller")
	}

	s.kubeSAM2MConfig(authDataStoreMutator, authDataStoreValidator)
}

func (s *datastorePostgresTestSuite) TestKubeSAM2MConfigPersistsAfterModification() {
	testMapping := storage.AuthMachineToMachineConfig_Mapping{
		Key:             "sub",
		Role:            testRole1,
		ValueExpression: "system:serviceaccount:my-namespace:my-service-account",
	}
	configControllerMapping := storage.AuthMachineToMachineConfig_Mapping{
		Key:             "sub",
		Role:            configController,
		ValueExpression: fmt.Sprintf("system:serviceaccount:%s:config-controller", env.Namespace.Setting()),
	}

	authDataStoreMutator := func(authDataStore DataStore) {
		var configs []*storage.AuthMachineToMachineConfig
		err := authDataStore.ForEachAuthM2MConfig(s.ctx, func(obj *storage.AuthMachineToMachineConfig) error {
			configs = append(configs, obj)
			return nil
		})
		s.NoError(err)
		kubeSAConfig := getTestConfig(configs)
		s.NotNil(kubeSAConfig)
		kubeSAConfig.Mappings = []*storage.AuthMachineToMachineConfig_Mapping{&testMapping}
		_, err = authDataStore.UpsertAuthM2MConfig(s.ctx, kubeSAConfig)
		s.NoError(err)
	}
	authDataStoreValidator := func(configs []*storage.AuthMachineToMachineConfig) {
		kubeSAConfig := getTestConfig(configs)
		s.NotNil(kubeSAConfig)
		s.Equal(2, len(kubeSAConfig.Mappings))
		for _, mapping := range []*storage.AuthMachineToMachineConfig_Mapping{&testMapping, &configControllerMapping} {
			found := false
			for _, kubeSAMapping := range kubeSAConfig.Mappings {
				fmt.Printf("key=%s; role=%s; valueExpression=%s\n", kubeSAMapping.Key, kubeSAMapping.Role, kubeSAMapping.ValueExpression)
				if kubeSAMapping.Key == mapping.Key && kubeSAMapping.Role == mapping.Role && kubeSAMapping.ValueExpression == mapping.ValueExpression {
					found = true
					break
				}
			}
			if !found {
				s.FailNowf("Failed to find role mapping", "key=%s; role=%s; valueExpression=%s", mapping.Key, mapping.Role, mapping.ValueExpression)
			}
		}
	}

	s.kubeSAM2MConfig(authDataStoreMutator, authDataStoreValidator)
}

func (s *datastorePostgresTestSuite) TestAddFKConstraint() {
	config, err := s.authDataStore.UpsertAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		Type:                    storage.AuthMachineToMachineConfig_GITHUB_ACTIONS,
		TokenExpirationDuration: "5m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "some-value",
				Role:            "non-existing-role",
			},
		},
	})
	s.ErrorIs(err, errox.ReferencedObjectNotFound)
	s.Nil(config)
}

func (s *datastorePostgresTestSuite) TestDeleteFKConstraint() {
	config, err := s.authDataStore.UpsertAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		Type:                    storage.AuthMachineToMachineConfig_GITHUB_ACTIONS,
		TokenExpirationDuration: "5m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "some-value",
				Role:            testRole1,
			},
		},
	})
	s.Require().NoError(err)

	s.ErrorIs(s.roleDataStore.RemoveRole(s.ctx, testRole1), errox.ReferencedByAnotherObject)

	s.NoError(s.authDataStore.RemoveAuthM2MConfig(s.ctx, config.GetId()))

	s.NoError(s.roleDataStore.RemoveRole(s.ctx, testRole1))
}

func (s *datastorePostgresTestSuite) TestAddUniqueIssuerConstraint() {
	_, err := s.authDataStore.UpsertAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		Type:                    storage.AuthMachineToMachineConfig_GENERIC,
		TokenExpirationDuration: "5m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "some-value",
				Role:            testRole1,
			},
		},
		Issuer: "https://stackrox.io",
	})

	s.NoError(err)

	_, err = s.authDataStore.UpsertAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		Id:                      "12c153c2-24a7-4b97-bd69-85b3a511241e",
		Type:                    storage.AuthMachineToMachineConfig_GENERIC,
		TokenExpirationDuration: "5m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "some-value",
				Role:            testRole2,
			},
		},
		Issuer: "https://stackrox.io",
	})

	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)
}

func (s *datastorePostgresTestSuite) addRoles(roleStore rolePostgresStore.Store) {
	permSetID := uuid.NewV4().String()
	accessScopeID := uuid.NewV4().String()
	s.Require().NoError(s.roleDataStore.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:          permSetID,
		Name:        "test permission set",
		Description: "test permission set",
		ResourceToAccess: map[string]storage.Access{
			resources.Access.String(): storage.Access_READ_ACCESS,
		},
	}))
	s.Require().NoError(s.roleDataStore.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
		Id:          accessScopeID,
		Name:        "test access scope",
		Description: "test access scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-a"},
		},
	}))

	for _, role := range testRoles.AsSlice() {
		s.Require().NoError(roleStore.Upsert(s.ctx, &storage.Role{
			Name:            role,
			Description:     "test role",
			PermissionSetId: permSetID,
			AccessScopeId:   accessScopeID,
		}))
	}
}
