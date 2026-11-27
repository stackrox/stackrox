package service

import (
	"context"
	"testing"

	backendMock "github.com/stackrox/rox/central/apitoken/backend/mocks"
	"github.com/stackrox/rox/central/role"
	roleMock "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestServiceImpl_ListAllowedRoles_SortsRoleAlphabetically(t *testing.T) {
	roleOne := roletest.NewResolvedRole("Writer", map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	roleTwo := roletest.NewResolvedRole("Admin", map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	roleThree := roletest.NewResolvedRole("Analyst", map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockIdentity := mocks.NewMockIdentity(mockCtrl)
	mockIdentity.EXPECT().Roles().Return([]permissions.ResolvedRole{roleOne, roleTwo, roleThree}).AnyTimes()

	mockDatastore := roleMock.NewMockDataStore(mockCtrl)
	mockDatastore.EXPECT().GetAllResolvedRoles(gomock.Any()).Return([]permissions.ResolvedRole{roleOne, roleTwo, roleThree}, nil)

	s := &serviceImpl{roles: mockDatastore}

	ctx := context.Background()
	ctx = authn.ContextWithIdentity(ctx, mockIdentity, t)

	actual, err := s.ListAllowedTokenRoles(ctx, &v1.Empty{})

	require.NoError(t, err)

	expected := []string{"Admin", "Analyst", "Writer"}
	assert.Equal(t, expected, actual.GetRoleNames())
}

// newMockIdentity returns a mock identity with the given roles that can be
// passed through authn.ContextWithIdentity. The methods required for token
// creation logging are stubbed out as well.
func newMockIdentity(ctrl *gomock.Controller, roles ...permissions.ResolvedRole) *mocks.MockIdentity {
	mockIdentity := mocks.NewMockIdentity(ctrl)
	mockIdentity.EXPECT().Roles().Return(roles).AnyTimes()
	mockIdentity.EXPECT().UID().Return("sso:0000-0000:admin").AnyTimes()
	mockIdentity.EXPECT().ExternalAuthProvider().Return(nil).AnyTimes()
	return mockIdentity
}

func TestServiceImpl_GenerateToken_RejectsInvalidRequests(t *testing.T) {
	writeRole := roletest.NewResolvedRole("Admin",
		map[string]storage.Access{
			"Image":      storage.Access_READ_WRITE_ACCESS,
			"Deployment": storage.Access_READ_WRITE_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	readRole := roletest.NewResolvedRole("Analyst",
		map[string]storage.Access{
			"Image":      storage.Access_READ_ACCESS,
			"Deployment": storage.Access_READ_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)

	testCases := map[string]struct {
		req        *v1.GenerateTokenRequest
		userRoles  []permissions.ResolvedRole
		prepareDS  func(ds *roleMock.MockDataStore)
		assertErr  require.ErrorAssertionFunc
		errorIsErr error
	}{
		"token name cannot be empty": {
			req:        &v1.GenerateTokenRequest{},
			assertErr:  require.Error,
			errorIsErr: errox.InvalidArgs,
		},
		"must use either role or roles, but not both": {
			req: &v1.GenerateTokenRequest{
				Name:  "test-token",
				Role:  "Analyst",
				Roles: []string{"Analyst"},
			},
			assertErr:  require.Error,
			errorIsErr: errox.InvalidArgs,
		},
		"requested role does not exist": {
			req: &v1.GenerateTokenRequest{
				Name: "test-token",
				Role: "DoesNotExist",
			},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "DoesNotExist").Return(nil, nil)
			},
			assertErr:  require.Error,
			errorIsErr: errox.InvalidArgs,
		},
		"role retrieval fails": {
			req: &v1.GenerateTokenRequest{
				Name: "test-token",
				Role: "Analyst",
			},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Analyst").Return(nil, assert.AnError)
			},
			assertErr: require.Error,
		},
		"privilege escalation is rejected": {
			req: &v1.GenerateTokenRequest{
				Name: "test-token",
				Role: "Admin",
			},
			userRoles: []permissions.ResolvedRole{readRole},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Admin").Return(writeRole, nil)
			},
			assertErr:  require.Error,
			errorIsErr: errox.NotAuthorized,
		},
		"no identity in context": {
			req: &v1.GenerateTokenRequest{
				Name: "test-token",
				Role: "Analyst",
			},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Analyst").Return(readRole, nil)
			},
			assertErr:  require.Error,
			errorIsErr: errox.NoCredentials,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockDatastore := roleMock.NewMockDataStore(mockCtrl)
			if tc.prepareDS != nil {
				tc.prepareDS(mockDatastore)
			}

			s := &serviceImpl{
				backend: backendMock.NewMockBackend(mockCtrl),
				roles:   mockDatastore,
			}

			ctx := context.Background()
			// The "no identity in context" case deliberately keeps a plain context.
			if name != "no identity in context" {
				ctx = authn.ContextWithIdentity(ctx, newMockIdentity(mockCtrl, tc.userRoles...), t)
			}

			resp, err := s.GenerateToken(ctx, tc.req)
			tc.assertErr(t, err)
			assert.Nil(t, resp)
			if tc.errorIsErr != nil {
				require.ErrorIs(t, err, tc.errorIsErr)
			}
		})
	}
}

func TestServiceImpl_GenerateToken_IssuesToken(t *testing.T) {
	writeRole := roletest.NewResolvedRole("Admin",
		map[string]storage.Access{
			"Image": storage.Access_READ_WRITE_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	readRole := roletest.NewResolvedRole("Analyst",
		map[string]storage.Access{
			"Image": storage.Access_READ_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	expiration := timestamppb.Now()

	expectedMetadata := &storage.TokenMetadata{
		Id:    "metadata-id",
		Name:  "test-token",
		Roles: []string{"Analyst"},
	}

	testCases := map[string]struct {
		req       *v1.GenerateTokenRequest
		prepareDS func(ds *roleMock.MockDataStore)
	}{
		"single role field is converted to roles": {
			req: &v1.GenerateTokenRequest{
				Name: "test-token",
				Role: "Analyst",
			},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Analyst").Return(readRole, nil)
			},
		},
		"multiple roles": {
			req: &v1.GenerateTokenRequest{
				Name:  "test-token",
				Roles: []string{"Analyst", "Admin"},
			},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Analyst").Return(readRole, nil)
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Admin").Return(writeRole, nil)
			},
		},
		"with expiration": {
			req: &v1.GenerateTokenRequest{
				Name:       "test-token",
				Role:       "Analyst",
				Expiration: expiration,
			},
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAndResolveRole(gomock.Any(), "Analyst").Return(readRole, nil)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockDatastore := roleMock.NewMockDataStore(mockCtrl)
			tc.prepareDS(mockDatastore)

			expectedRoleNames := []string{"Analyst"}
			if name == "multiple roles" {
				expectedRoleNames = []string{"Analyst", "Admin"}
			}
			// The user holds the Admin role, hence requesting any subset of its
			// permissions must not be considered a privilege escalation.
			mockBackend := backendMock.NewMockBackend(mockCtrl)
			mockBackend.EXPECT().IssueRoleToken(gomock.Any(), "test-token", expectedRoleNames, gomock.Any()).
				Return("issued-token", expectedMetadata, nil)

			s := &serviceImpl{
				backend: mockBackend,
				roles:   mockDatastore,
			}

			ctx := authn.ContextWithIdentity(context.Background(), newMockIdentity(mockCtrl, writeRole), t)

			resp, err := s.GenerateToken(ctx, tc.req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, "issued-token", resp.GetToken())
			assert.Equal(t, expectedMetadata, resp.GetMetadata())
		})
	}
}

func TestServiceImpl_ListAllowedTokenRoles_FiltersUnassignableRoles(t *testing.T) {
	readRole := roletest.NewResolvedRole("Analyst",
		map[string]storage.Access{
			"Image": storage.Access_READ_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	writeRole := roletest.NewResolvedRole("Admin",
		map[string]storage.Access{
			"Image": storage.Access_READ_WRITE_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	noneRole := roletest.NewResolvedRole(accesscontrol.None,
		map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDatastore := roleMock.NewMockDataStore(mockCtrl)
	mockDatastore.EXPECT().GetAllResolvedRoles(gomock.Any()).
		Return([]permissions.ResolvedRole{writeRole, noneRole, readRole}, nil)

	s := &serviceImpl{roles: mockDatastore}

	ctx := authn.ContextWithIdentity(context.Background(), newMockIdentity(mockCtrl, readRole), t)

	actual, err := s.ListAllowedTokenRoles(ctx, &v1.Empty{})

	require.NoError(t, err)
	// "None" is never assignable and "Admin" would be a privilege escalation
	// for a user that only holds the "Analyst" role.
	assert.Equal(t, []string{"Analyst"}, actual.GetRoleNames())
}

func TestServiceImpl_ListAllowedTokenRoles_ErrorCases(t *testing.T) {
	testCases := map[string]struct {
		prepareDS func(ds *roleMock.MockDataStore)
		withID    bool
		errorIs   error
	}{
		"role retrieval fails": {
			prepareDS: func(ds *roleMock.MockDataStore) {
				ds.EXPECT().GetAllResolvedRoles(gomock.Any()).Return(nil, assert.AnError)
			},
			withID:  true,
			errorIs: assert.AnError,
		},
		"no identity in context": {
			withID:  false,
			errorIs: errox.NoCredentials,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockDatastore := roleMock.NewMockDataStore(mockCtrl)
			if tc.prepareDS != nil {
				tc.prepareDS(mockDatastore)
			}

			s := &serviceImpl{roles: mockDatastore}

			ctx := context.Background()
			if tc.withID {
				ctx = authn.ContextWithIdentity(ctx, newMockIdentity(mockCtrl), t)
			}

			resp, err := s.ListAllowedTokenRoles(ctx, &v1.Empty{})

			require.ErrorIs(t, err, tc.errorIs)
			assert.Nil(t, resp)
		})
	}
}
