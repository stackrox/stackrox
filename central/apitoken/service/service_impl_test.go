package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/role"
	roleMock "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSomethingIsSorted(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockIdentity := mocks.NewMockIdentity(mockCtrl)

	mockDatastore := roleMock.NewMockDataStore(mockCtrl)
	roleOne := roletest.NewResolvedRole("Analyst", map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	roleTwo := roletest.NewResolvedRole("Admin", map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	roleThree := roletest.NewResolvedRole("Writer", map[string]storage.Access{},
		role.AccessScopeIncludeAll,
	)

	mockIdentity.EXPECT().Roles().Return([]permissions.ResolvedRole{roleOne, roleTwo, roleThree}).AnyTimes()

	mockDatastore.EXPECT().GetAllResolvedRoles(gomock.Any()).Return([]permissions.ResolvedRole{roleOne, roleTwo, roleThree}, nil)

	s := &serviceImpl{roles: mockDatastore}

	ctx := context.Background()
	ctx = authn.ContextWithIdentity(ctx, mockIdentity, t)

	actual, err := s.ListAllowedTokenRoles(ctx, &v1.Empty{})

	assert.NoError(t, err)

	expected := []string{"Admin", "Analyst", "Writer"}
	assert.Equal(t, expected, actual.RoleNames)
}
