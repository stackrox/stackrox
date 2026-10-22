package service

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/apitoken/backend"
	"github.com/stackrox/rox/central/role"
	roleMock "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type recordingTokenBackend struct {
	backend.Backend
	scopes []tokens.HTTPRequestScope
}

func (b *recordingTokenBackend) IssueRoleToken(_ context.Context, _ string, _ []string, _ *time.Time, scopes []tokens.HTTPRequestScope) (string, *storage.TokenMetadata, error) {
	b.scopes = scopes
	return "token", &storage.TokenMetadata{}, nil
}

func TestGenerateTokenHTTPRequestScope(t *testing.T) {
	role := roletest.NewResolvedRole("Analyst", map[string]storage.Access{}, role.AccessScopeIncludeAll)
	ctrl := gomock.NewController(t)
	roles := roleMock.NewMockDataStore(ctrl)
	roles.EXPECT().GetAndResolveRole(gomock.Any(), "Analyst").AnyTimes().Return(role, nil)
	identity := mocks.NewMockIdentity(ctrl)
	identity.EXPECT().Roles().AnyTimes().Return([]permissions.ResolvedRole{role})
	identity.EXPECT().UID().AnyTimes().Return("user")
	identity.EXPECT().ExternalAuthProvider().AnyTimes().Return(nil)
	ctx := authn.ContextWithIdentity(t.Context(), identity, t)
	b := &recordingTokenBackend{}
	svc := &serviceImpl{roles: roles, backend: b}

	result, err := svc.GenerateToken(ctx, &v1.GenerateTokenRequest{
		Name: "test", Roles: []string{"Analyst"},
		HttpRequestScopes: []*v1.HTTPRequestScope{{Method: "GET", Path: "/v1/metadata"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "token", result.GetToken())
	assert.Equal(t, []tokens.HTTPRequestScope{{Method: "GET", Path: "/v1/metadata"}}, b.scopes)

	_, err = svc.GenerateToken(ctx, &v1.GenerateTokenRequest{
		Name: "test", Roles: []string{"Analyst"},
		HttpRequestScopes: []*v1.HTTPRequestScope{{Method: "GET", Path: "/v1/%6detadata"}},
	})
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

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
