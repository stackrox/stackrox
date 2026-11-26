package observe

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestUtils(t *testing.T) {
	suite.Run(t, new(utilsTestSuite))
}

type utilsTestSuite struct {
	suite.Suite
}

func (s *utilsTestSuite) TestCalculateUser() {
	ctx := context.Background()

	s.Run("without identity in context", func() {
		s.Nil(calculateUser(ctx), "user should be nil if context has no identity")
	})

	s.Run("with identity in context", func() {
		ctrl := gomock.NewController(s.T())

		accessScope := &storage.SimpleAccessScope{
			Name: "scope-name",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{ClusterName: "cluster-name", NamespaceName: "namespace"},
				},
			},
		}

		role := mocks.NewMockResolvedRole(ctrl)
		role.EXPECT().GetRoleName().Return("Admin")
		role.EXPECT().GetPermissions().Return(map[string]storage.Access{
			"Alarm": storage.Access_READ_WRITE_ACCESS,
		})
		// GetAccessScope() is called both for the scope name and the scope rules.
		role.EXPECT().GetAccessScope().Return(accessScope).AnyTimes()

		identity := authnMocks.NewMockIdentity(ctrl)
		// User() is called both for the username and the friendly name,
		// Roles() both for the slice capacity and the iteration.
		identity.EXPECT().User().Return(&storage.UserInfo{
			Username:     "user-name",
			FriendlyName: "friendly-name",
		}).AnyTimes()
		identity.EXPECT().Permissions().Return(map[string]storage.Access{
			"Alarm": storage.Access_READ_WRITE_ACCESS,
		}).AnyTimes()
		identity.EXPECT().Roles().Return([]permissions.ResolvedRole{role}).AnyTimes()

		ctxWithIdentity := authn.ContextWithIdentity(ctx, identity, s.T())
		user := calculateUser(ctxWithIdentity)

		s.Require().NotNil(user)
		s.Equal("user-name", user.GetUsername())
		s.Equal("friendly-name", user.GetFriendlyName())
		s.Equal(map[string]storage.Access{
			"Alarm": storage.Access_READ_WRITE_ACCESS,
		}, user.GetAggregatedPermissions())
		s.Require().Len(user.GetRoles(), 1)
		s.Equal("Admin", user.GetRoles()[0].GetName())
		s.Equal(map[string]storage.Access{
			"Alarm": storage.Access_READ_WRITE_ACCESS,
		}, user.GetRoles()[0].GetPermissions())
		s.Equal("scope-name", user.GetRoles()[0].GetAccessScopeName())
		s.Equal(accessScope.GetRules(), user.GetRoles()[0].GetAccessScope())
	})
}
