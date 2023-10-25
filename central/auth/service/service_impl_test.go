//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/auth/datastore"
	pgStore "github.com/stackrox/rox/central/auth/store/postgres"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	permissionSetPostgresStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	rolePostgresStore "github.com/stackrox/rox/central/role/store/role/postgres"
	accessScopePostgresStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	targetEndPointName = "/v1.AuthService/GetAuthStatus"

	testRole1 = "New-Admin"
	testRole2 = "Super-Admin"
	testRole3 = "Super Continuous Integration"
)

var (
	testRoles = set.NewFrozenStringSet(testRole1, testRole2, testRole3)
)

func TestAuthServiceAccessControl(t *testing.T) {
	suite.Run(t, new(authServiceAccessControlTestSuite))
}

type authServiceAccessControlTestSuite struct {
	suite.Suite

	svc    *serviceImpl
	roleDS roleDataStore.DataStore

	pool *pgtest.TestPostgres

	authProvider authproviders.Provider

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context

	accessCtx context.Context
}

func (s *authServiceAccessControlTestSuite) SetupSuite() {
	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)
	s.authProvider = authProvider
	s.withAdminRoleCtx = basic.ContextWithAdminIdentity(s.T(), s.authProvider)
	s.withNoneRoleCtx = basic.ContextWithNoneIdentity(s.T(), s.authProvider)
	s.withNoAccessCtx = basic.ContextWithNoAccessIdentity(s.T(), s.authProvider)
	s.withNoRoleCtx = basic.ContextWithNoRoleIdentity(s.T(), s.authProvider)
	s.anonymousCtx = context.Background()

	s.accessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access),
		),
	)
}

func (s *authServiceAccessControlTestSuite) SetupTest() {
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)

	store := pgStore.New(s.pool.DB)
	authDataStore := datastore.New(store)

	permSetStore := permissionSetPostgresStore.New(s.pool.DB)
	accessScopeStore := accessScopePostgresStore.New(s.pool.DB)
	roleStore := rolePostgresStore.New(s.pool.DB)
	s.roleDS = roleDataStore.New(roleStore, permSetStore, accessScopeStore, func(_ context.Context, _ func(*storage.Group) bool) ([]*storage.Group, error) {
		return nil, nil
	})

	s.addRoles()

	s.svc = &serviceImpl{authDataStore: authDataStore}
}

func (s *authServiceAccessControlTestSuite) TearDownTest() {
	s.pool.Teardown(s.T())
	s.pool.Close()
}

type testCase struct {
	name string
	ctx  context.Context

	expectedAuthorizerError error
	expectedServiceError    error
}

func (s *authServiceAccessControlTestSuite) getTestCases() []testCase {
	return []testCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedServiceError:    errox.NoCredentials,
			expectedAuthorizerError: nil,
		},
	}
}

func (s *authServiceAccessControlTestSuite) TestAuthServiceAuthorizer() {
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			ctx, err := s.svc.AuthFuncOverride(c.ctx, targetEndPointName)
			s.ErrorIs(err, c.expectedAuthorizerError)
			s.Equal(c.ctx, ctx)
		})
	}
}

func (s *authServiceAccessControlTestSuite) TestAuthServiceResponse() {
	emptyQuery := &v1.Empty{}
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.GetAuthStatus(c.ctx, emptyQuery)
			s.ErrorIs(err, c.expectedServiceError)
			if c.expectedServiceError == nil {
				s.NotNil(rsp)
				s.Equal(c.name, rsp.GetUserInfo().GetUsername())
				s.Equal(uuid.NewDummy().String(), rsp.GetAuthProvider().GetId())
			} else {
				s.Nil(rsp)
			}
		})
	}
}

func (s *authServiceAccessControlTestSuite) TestValidateAuthMachineToMachineConfig() {
	testCases := map[string]struct {
		config      *v1.AuthMachineToMachineConfig
		skipIDCheck bool
		err         error
	}{
		"nil config": {
			config: nil,
			err:    errox.InvalidArgs,
		},
		"empty ID given and ID validation is not skipped": {
			config: &v1.AuthMachineToMachineConfig{
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errEmptyID,
		},
		"invalid token expiration - parsing duration": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidTokenExpiration,
		},
		"invalid token expiration - duration is empty": {
			config: &v1.AuthMachineToMachineConfig{
				Id:     "some-id",
				Type:   v1.AuthMachineToMachineConfig_GENERIC,
				Issuer: "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidTokenExpiration,
		},
		"invalid token expiration - duration is too low": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "1s",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidTokenExpiration,
		},
		"invalid token expiration - duration is too high": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "24h1s",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidTokenExpiration,
		},
		"invalid issuer - empty issuer for GENERIC type": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidIssuer,
		},
		"invalid issuer - URL cannot be parsed": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://something-invalid/%+o",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidIssuer,
		},
		"invalid regular expression - parsing the expression": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "a(b",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidRegularExpression,
		},
		"invalid regular expression - empty regular expression": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidRegularExpression,
		},
		"invalid issuer - non-github actions issuer for type GitHub actions": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
			err: errInvalidIssuer,
		},
		"valid config for GENERIC": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
		},
		"valid config for GITHUB_ACTIONS with empty issuer": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
		},
		"valid config for GITHUB_ACTIONS with issuer set": {
			config: &v1.AuthMachineToMachineConfig{
				Id:                      "some-id",
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
				Issuer:                  githubActionsIssuer,
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
		},
		"valid config without ID but skipping the ID validation": {
			skipIDCheck: true,
			config: &v1.AuthMachineToMachineConfig{
				TokenExpirationDuration: "5m",
				Type:                    v1.AuthMachineToMachineConfig_GENERIC,
				Issuer:                  "https://stackrox.io",
				Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
					{
						Key:             "some-key",
						ValueExpression: "some-value",
						Role:            testRole1,
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			err := s.svc.validateAuthMachineToMachineConfig(testCase.config, testCase.skipIDCheck)
			s.ErrorIs(err, testCase.err)
		})
	}
}

func (s *authServiceAccessControlTestSuite) TestGetConfig() {
	addConfigResp, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: &v1.AuthMachineToMachineConfig{
			TokenExpirationDuration: "1h",
			Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something",
					Role:            testRole1,
				},
				{
					Key:             "aud",
					ValueExpression: "github",
					Role:            testRole3,
				},
			},
		},
	})
	s.Require().NoError(err)

	getConfigResp, err := s.svc.GetAuthMachineToMachineConfig(s.accessCtx,
		&v1.ResourceByID{Id: addConfigResp.GetConfig().GetId()})
	s.NoError(err)
	s.Equal(addConfigResp.GetConfig(), getConfigResp.GetConfig())
}

func (s *authServiceAccessControlTestSuite) TestGetConfigNonExisting() {
	getConfigResp, err := s.svc.GetAuthMachineToMachineConfig(s.accessCtx,
		&v1.ResourceByID{Id: "80c053c2-24a7-4b97-bd69-85b3a511241e"})
	s.ErrorIs(err, errox.NotFound)
	s.Nil(getConfigResp)
}

func (s *authServiceAccessControlTestSuite) TestAddGitHubActionsConfig() {
	config := &v1.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
	}

	resp, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: config,
	})
	s.NoError(err)
	s.NotEmpty(resp.GetConfig().GetId())
}

func (s *authServiceAccessControlTestSuite) TestAddGenericConfig() {
	config := &v1.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1h",
		Type:                    v1.AuthMachineToMachineConfig_GENERIC,
		Issuer:                  "https://stackrox.io",
	}

	resp, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: config,
	})
	s.NoError(err)
	s.NotEmpty(resp.GetConfig().GetId())
}

func (s *authServiceAccessControlTestSuite) TestAddGitHubActionsConfigWithID() {
	config := &v1.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		TokenExpirationDuration: "1h",
		Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
	}

	resp, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: config,
	})
	s.NoError(err)
	s.NotEmpty(resp.GetConfig().GetId())
	s.NotEqual(resp.GetConfig().GetId(), "80c053c2-24a7-4b97-bd69-85b3a511241e")
}

func (s *authServiceAccessControlTestSuite) TestAddGenericConfigWithID() {
	config := &v1.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		TokenExpirationDuration: "1h",
		Type:                    v1.AuthMachineToMachineConfig_GENERIC,
		Issuer:                  "https://stackrox.io",
	}

	resp, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: config,
	})
	s.NoError(err)
	s.NotEmpty(resp.GetConfig().GetId())
	s.NotEqual(resp.GetConfig().GetId(), "80c053c2-24a7-4b97-bd69-85b3a511241e")
}

func (s *authServiceAccessControlTestSuite) TestListConfigs() {
	config1, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: &v1.AuthMachineToMachineConfig{
			TokenExpirationDuration: "1h",
			Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something",
					Role:            testRole1,
				},
				{
					Key:             "aud",
					ValueExpression: "github",
					Role:            testRole3,
				},
			},
		},
	})
	s.Require().NoError(err)

	config2, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: &v1.AuthMachineToMachineConfig{
			TokenExpirationDuration: "1h",
			Type:                    v1.AuthMachineToMachineConfig_GENERIC,
			Issuer:                  "https://stackrox.io",
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "somewhere",
					Role:            testRole1,
				},
				{
					Key:             "aud",
					ValueExpression: "the",
					Role:            testRole2,
				},
			},
		},
	})
	s.Require().NoError(err)

	configs, err := s.svc.ListAuthMachineToMachineConfigs(s.accessCtx, nil)
	s.NoError(err)

	s.ElementsMatch(configs.GetConfigs(), []*v1.AuthMachineToMachineConfig{config1.GetConfig(), config2.GetConfig()})
}

func (s *authServiceAccessControlTestSuite) TestUpdateExistingConfig() {
	config, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: &v1.AuthMachineToMachineConfig{
			TokenExpirationDuration: "1h",
			Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something",
					Role:            testRole1,
				},
				{
					Key:             "aud",
					ValueExpression: "github",
					Role:            testRole3,
				},
			},
		},
	})
	s.Require().NoError(err)

	config.GetConfig().Mappings = []*v1.AuthMachineToMachineConfig_Mapping{
		{
			Key:             "sub",
			ValueExpression: "someone",
			Role:            testRole2,
		},
	}

	_, err = s.svc.UpdateAuthMachineToMachineConfig(s.accessCtx,
		&v1.UpdateAuthMachineToMachineConfigRequest{Config: config.GetConfig()})
	s.NoError(err)

	updatedConfig, err := s.svc.GetAuthMachineToMachineConfig(s.accessCtx, &v1.ResourceByID{Id: config.GetConfig().GetId()})
	s.NoError(err)

	s.Equal(config.GetConfig(), updatedConfig.GetConfig())
}

func (s *authServiceAccessControlTestSuite) TestUpdateAddConfig() {
	newConfig := &v1.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		TokenExpirationDuration: "1m",
		Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
	}
	_, err := s.svc.UpdateAuthMachineToMachineConfig(s.accessCtx, &v1.UpdateAuthMachineToMachineConfigRequest{
		Config: newConfig,
	})
	s.NoError(err)
}

func (s *authServiceAccessControlTestSuite) TestUpdateConfigWithEmptyID() {
	newConfig := &v1.AuthMachineToMachineConfig{
		TokenExpirationDuration: "1m",
		Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
	}
	_, err := s.svc.UpdateAuthMachineToMachineConfig(s.accessCtx, &v1.UpdateAuthMachineToMachineConfigRequest{
		Config: newConfig,
	})
	s.ErrorIs(err, errEmptyID)
}

func (s *authServiceAccessControlTestSuite) TestRemoveConfig() {
	config, err := s.svc.AddAuthMachineToMachineConfig(s.accessCtx, &v1.AddAuthMachineToMachineConfigRequest{
		Config: &v1.AuthMachineToMachineConfig{
			TokenExpirationDuration: "1h",
			Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something",
					Role:            testRole1,
				},
				{
					Key:             "aud",
					ValueExpression: "github",
					Role:            testRole3,
				},
			},
		},
	})
	s.Require().NoError(err)

	_, err = s.svc.DeleteAuthMachineToMachineConfig(s.accessCtx, &v1.ResourceByID{Id: config.GetConfig().GetId()})
	s.NoError(err)

	configResponse, err := s.svc.GetAuthMachineToMachineConfig(s.accessCtx,
		&v1.ResourceByID{Id: config.GetConfig().GetId()})
	s.ErrorIs(err, errox.NotFound)
	s.Nil(configResponse)
}

func (s *authServiceAccessControlTestSuite) TestRemoveNonExistingConfig() {
	_, err := s.svc.DeleteAuthMachineToMachineConfig(s.accessCtx, &v1.ResourceByID{Id: "80c053c2-24a7-4b97-bd69-85b3a511241e"})
	s.NoError(err)
}

func (s *authServiceAccessControlTestSuite) addRoles() {
	permSetID := uuid.NewV4().String()
	accessScopeID := uuid.NewV4().String()
	s.Require().NoError(s.roleDS.AddPermissionSet(s.accessCtx, &storage.PermissionSet{
		Id:          permSetID,
		Name:        "test permission set",
		Description: "test permission set",
		ResourceToAccess: map[string]storage.Access{
			resources.Access.String(): storage.Access_READ_ACCESS,
		},
	}))
	s.Require().NoError(s.roleDS.AddAccessScope(s.accessCtx, &storage.SimpleAccessScope{
		Id:          accessScopeID,
		Name:        "test access scope",
		Description: "test access scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-a"},
		},
	}))

	for _, role := range testRoles.AsSlice() {
		s.Require().NoError(s.roleDS.AddRole(s.accessCtx, &storage.Role{
			Name:            role,
			Description:     "test role",
			PermissionSetId: permSetID,
			AccessScopeId:   accessScopeID,
		}))
	}
}
