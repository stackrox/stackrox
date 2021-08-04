package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/role/resources"
	storeMocks "github.com/stackrox/rox/central/sac/datastore/internal/store/mocks"
	sacMocks "github.com/stackrox/rox/central/sac/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

// Test the logic of updating configs.
//////////////////////////////////////
func TestAuthzConfigDatatStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(authzDataStoreTestSuite))
}

type authzDataStoreTestSuite struct {
	suite.Suite

	hasWriteCtx context.Context

	mockCtrl          *gomock.Controller
	mockStorage       *storeMocks.MockStore
	mockClientManager *sacMocks.MockAuthPluginClientManger
}

func (s *authzDataStoreTestSuite) SetupTest() {
	s.hasWriteCtx = WithModifyEnabledPluginCap(sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.AuthPlugin))))

	s.mockCtrl = gomock.NewController(s.T())
	s.mockStorage = storeMocks.NewMockStore(s.mockCtrl)
	s.mockClientManager = sacMocks.NewMockAuthPluginClientManger(s.mockCtrl)
}

func (s *authzDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authzDataStoreTestSuite) getDataStore(initialContents []*storage.AuthzPluginConfig) DataStore {
	dataStore := &datastoreImpl{
		storage:   s.mockStorage,
		clientMgr: s.mockClientManager,
	}
	s.mockStorage.EXPECT().ListAuthzPluginConfigs().Return(initialContents, nil)
	s.mockClientManager.EXPECT().SetClient(gomock.Any())
	err := dataStore.Initialize()
	s.NoError(err)
	return dataStore
}

func (s *authzDataStoreTestSuite) TestMultipleEnabledRecovery() {
	current := []*storage.AuthzPluginConfig{
		{
			Id:      "id1",
			Enabled: true,
			EndpointConfig: &storage.HTTPEndpointConfig{
				Endpoint: "https://test",
			},
		},
		{
			Id:      "id2",
			Enabled: true,
			EndpointConfig: &storage.HTTPEndpointConfig{
				Endpoint: "https://test",
			},
		},
	}
	disabled := current[1].Clone()
	disabled.Enabled = false
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(disabled).Return(nil)
	s.getDataStore(current)
}

func (s *authzDataStoreTestSuite) TestUpsertNewEnabled() {
	// We should fetch all current configs to search for any enabled.
	current := []*storage.AuthzPluginConfig{
		{
			Id:      "id1",
			Enabled: false,
		},
		{
			Id:      "id2",
			Enabled: false,
		},
	}

	// We should look up the new config to see if the ID exists before updating it
	s.mockStorage.EXPECT().GetAuthzPluginConfig(current[1].GetId()).Return(current[1], nil)

	// We should upsert the new config.
	upserted := &storage.AuthzPluginConfig{
		Id:      "id2",
		Enabled: true,
		EndpointConfig: &storage.HTTPEndpointConfig{
			Endpoint: "https://endpoint",
		},
	}
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(upserted).Return(nil)

	// Since the new is enabled, we expect a call to swap out the current client.
	s.mockClientManager.EXPECT().SetClient(gomock.Any())

	returnedConfig, err := s.getDataStore(current).UpsertAuthzPluginConfig(s.hasWriteCtx, upserted)
	s.NoError(err, "expected no error trying to write with permissions")
	s.Equal(upserted, returnedConfig)
}

func (s *authzDataStoreTestSuite) TestEditEnabledPlugin() {
	current := []*storage.AuthzPluginConfig{
		{
			Id:      "id1",
			Enabled: false,
		},
		{
			Id:      "id2",
			Enabled: true,
			EndpointConfig: &storage.HTTPEndpointConfig{
				Endpoint: "https://test",
			},
		},
	}
	modifiedCurrentlyEnabled := current[1].Clone()
	modifiedCurrentlyEnabled.EndpointConfig = &storage.HTTPEndpointConfig{Endpoint: "https://AnotherEndpoint"}

	s.mockStorage.EXPECT().GetAuthzPluginConfig(current[1].GetId()).Return(current[1], nil)
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(modifiedCurrentlyEnabled).Return(nil)
	s.mockClientManager.EXPECT().SetClient(gomock.Any())

	returnedConfig, err := s.getDataStore(current).UpsertAuthzPluginConfig(s.hasWriteCtx, modifiedCurrentlyEnabled)
	s.NoError(err)
	s.Equal(returnedConfig, modifiedCurrentlyEnabled)
}

func (s *authzDataStoreTestSuite) TestUpsertNewDisabled() {
	// We should only upsert the new config. Since it is disabled, it doesn't have any side effects.
	// New configs will be assigned IDs by the datastore
	upserted := &storage.AuthzPluginConfig{
		Enabled: false,
	}
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(upserted).Return(nil)

	returnedConfig, err := s.getDataStore(nil).UpsertAuthzPluginConfig(s.hasWriteCtx, upserted)
	s.NoError(err, "expected no error trying to write with permissions")
	// Checking s.Equal is convenient but misleading.  returnedConfig will not be guaranteed to be the same object as
	// upserted and returnedConfig will have it's ID set.
	s.Equal(returnedConfig, upserted)
	// ID should have been set
	s.NotEqual("", returnedConfig.GetId())
}

func (s *authzDataStoreTestSuite) TestUpsertCurrentEnabled() {
	// We should fetch all current configs to search for any enabled.
	current := []*storage.AuthzPluginConfig{
		{
			Id:      "id1",
			Enabled: false,
		},
		{
			Id:      "id2",
			Enabled: true,
			EndpointConfig: &storage.HTTPEndpointConfig{
				Endpoint: "https://test",
			},
		},
	}

	// The currently enabled config should be stored as disabled.
	modifiedCurrentlyEnabled := current[1].Clone()
	modifiedCurrentlyEnabled.Enabled = false
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(modifiedCurrentlyEnabled).Return(nil)

	// We should upsert the new config.
	upserted := &storage.AuthzPluginConfig{
		Enabled: true,
		EndpointConfig: &storage.HTTPEndpointConfig{
			Endpoint: "https://endpoint",
		},
	}
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(upserted).Return(nil)

	// Since the new is enabled, we expect a call to swap out the current client.
	s.mockClientManager.EXPECT().SetClient(gomock.Any())

	returnedConfig, err := s.getDataStore(current).UpsertAuthzPluginConfig(s.hasWriteCtx, upserted)
	s.NoError(err, "expected no error trying to write with permissions")
	// Checking s.Equal is convenient but misleading.  returnedConfig will not be guaranteed to be the same object as
	// upserted and returnedConfig will have it's ID set.
	s.Equal(upserted, returnedConfig)
}

func (s *authzDataStoreTestSuite) TestDeleteDisabled() {
	// Since the config is disabled, it should just get deletes with no side effects.
	s.mockStorage.EXPECT().DeleteAuthzPluginConfig("id").Return(nil)

	err := s.getDataStore(nil).DeleteAuthzPluginConfig(s.hasWriteCtx, "id")
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authzDataStoreTestSuite) TestDeleteEnabled() {
	// We should list the configs in order to initialize the currently enabled config cache
	current := []*storage.AuthzPluginConfig{
		{
			Id:      "id",
			Enabled: true,
			EndpointConfig: &storage.HTTPEndpointConfig{
				Endpoint: "https://endpoint1",
			},
		},
	}

	// The config should be deleted.
	s.mockStorage.EXPECT().DeleteAuthzPluginConfig("id").Return(nil)

	// Since the config is enabled, we should remove the client from auth.
	s.mockClientManager.EXPECT().SetClient(gomock.Nil())

	err := s.getDataStore(current).DeleteAuthzPluginConfig(s.hasWriteCtx, "id")
	s.NoError(err, "expected no error trying to write with permissions")
}

// Test that scoped access control is enforced on the datastore.
////////////////////////////////////////////////////////////////
func TestAuthzConfigDatatStoreAccess(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(authzDataStoreAccessTestSuite))
}

type authzDataStoreAccessTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockCtrl    *gomock.Controller
	mockStorage *storeMocks.MockStore
}

func (s *authzDataStoreAccessTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.AuthPlugin)))
	s.hasWriteCtx = WithModifyEnabledPluginCap(sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.AuthPlugin))))

	s.mockCtrl = gomock.NewController(s.T())
	s.mockStorage = storeMocks.NewMockStore(s.mockCtrl)
}

func (s *authzDataStoreAccessTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authzDataStoreAccessTestSuite) getDataStore() DataStore {
	return &datastoreImpl{
		storage: s.mockStorage,
	}
}

func (s *authzDataStoreAccessTestSuite) TestEnforcesList() {
	s.mockStorage.EXPECT().ListAuthzPluginConfigs().Times(0)

	configs, err := s.getDataStore().ListAuthzPluginConfigs(s.hasNoneCtx)
	s.NoError(err, "expected no error")
	s.Nil(configs, "expected return value to be nil")
}

func (s *authzDataStoreAccessTestSuite) TestEnforcesGet() {
	s.mockStorage.EXPECT().GetAuthzPluginConfig("id").Times(0)

	plugin, err := s.getDataStore().GetAuthzPluginConfig(s.hasNoneCtx, "id")
	s.Error(err, "expected permission denied error")
	s.Nil(plugin, "expected return value to be nil")
}

func (s *authzDataStoreAccessTestSuite) TestAllowsList() {
	dataStore := s.getDataStore()
	s.mockStorage.EXPECT().ListAuthzPluginConfigs().Return(nil, nil)

	_, err := dataStore.ListAuthzPluginConfigs(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.mockStorage.EXPECT().ListAuthzPluginConfigs().Return(nil, nil)

	_, err = dataStore.ListAuthzPluginConfigs(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *authzDataStoreAccessTestSuite) TestAllowsGet() {
	storedPlugin := &storage.AuthzPluginConfig{Id: "id"}
	dataStore := s.getDataStore()
	s.mockStorage.EXPECT().GetAuthzPluginConfig(storedPlugin.GetId()).Return(storedPlugin, nil)

	plugin, err := dataStore.GetAuthzPluginConfig(s.hasReadCtx, storedPlugin.GetId())
	s.Equal(plugin, storedPlugin)
	s.NoError(err, "expected no error trying to read with permissions")

	s.mockStorage.EXPECT().GetAuthzPluginConfig(storedPlugin.GetId()).Return(storedPlugin, nil)

	plugin, err = dataStore.GetAuthzPluginConfig(s.hasWriteCtx, storedPlugin.GetId())
	s.Equal(plugin, storedPlugin)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *authzDataStoreAccessTestSuite) TestEnforcesUpsert() {
	dataStore := s.getDataStore()
	upserted := &storage.AuthzPluginConfig{
		Id:      "id",
		Enabled: false,
	}

	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(upserted).Times(0)

	_, err := dataStore.UpsertAuthzPluginConfig(s.hasNoneCtx, upserted)
	s.Error(err, "expected an error trying to write without permissions")

	_, err = dataStore.UpsertAuthzPluginConfig(s.hasReadCtx, upserted)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *authzDataStoreAccessTestSuite) TestAllowsUpsert() {
	upserted := &storage.AuthzPluginConfig{
		Id:      "id",
		Enabled: false,
	}

	s.mockStorage.EXPECT().GetAuthzPluginConfig(upserted.GetId()).Return(upserted, nil)
	s.mockStorage.EXPECT().UpsertAuthzPluginConfig(upserted).Return(nil)

	_, err := s.getDataStore().UpsertAuthzPluginConfig(s.hasWriteCtx, upserted)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authzDataStoreAccessTestSuite) TestEnforcesDelete() {
	dataStore := s.getDataStore()
	s.mockStorage.EXPECT().DeleteAuthzPluginConfig(gomock.Any()).Times(0)

	err := dataStore.DeleteAuthzPluginConfig(s.hasNoneCtx, "id")
	s.Error(err, "expected an error trying to write without permissions")

	err = dataStore.DeleteAuthzPluginConfig(s.hasReadCtx, "id")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *authzDataStoreAccessTestSuite) TestAllowsDelete() {
	s.mockStorage.EXPECT().DeleteAuthzPluginConfig("id").Return(nil)

	err := s.getDataStore().DeleteAuthzPluginConfig(s.hasWriteCtx, "id")
	s.NoError(err, "expected no error trying to write with permissions")
}
