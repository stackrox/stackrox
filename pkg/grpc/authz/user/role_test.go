package user

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWithRole_Authorized(t *testing.T) {
	const requiredRole = "Admin"

	makeCtx := func(it *testing.T, roles ...permissions.ResolvedRole) context.Context {
		ctrl := gomock.NewController(it)
		id := mocks.NewMockIdentity(ctrl)
		id.EXPECT().Roles().Return(roles).AnyTimes()
		return authn.ContextWithIdentity(it.Context(), id, it)
	}

	tests := map[string]struct {
		ctx     context.Context
		wantErr error
	}{
		"no identity in context": {
			ctx:     context.Background(),
			wantErr: errox.NoCredentials,
		},
		"identity with no roles": {
			ctx:     makeCtx(t),
			wantErr: auth.ErrNoValidRole,
		},
		"identity with matching role": {
			ctx: makeCtx(t, roletest.NewResolvedRoleWithDenyAll(requiredRole, nil)),
		},
		"identity with non-matching role": {
			ctx:     makeCtx(t, roletest.NewResolvedRoleWithDenyAll("Analyst", nil)),
			wantErr: errox.NotAuthorized,
		},
		"identity with multiple roles, one matches": {
			ctx: makeCtx(t,
				roletest.NewResolvedRoleWithDenyAll("Analyst", nil),
				roletest.NewResolvedRoleWithDenyAll(requiredRole, nil),
			),
		},
		"identity with multiple roles, none match": {
			ctx: makeCtx(t,
				roletest.NewResolvedRoleWithDenyAll("Analyst", nil),
				roletest.NewResolvedRoleWithDenyAll("Vuln Manager", nil),
			),
			wantErr: errox.NotAuthorized,
		},
		"method name is ignored": {
			ctx: makeCtx(t, roletest.NewResolvedRoleWithDenyAll(requiredRole, nil)),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			err := WithRole(requiredRole).Authorized(tt.ctx, "/some.Service/Method")
			assert.ErrorIs(it, err, tt.wantErr)
		})
	}
}

func TestRoleChecker_checkRole(t *testing.T) {
	const requiredRole = "Admin"
	checker := &roleChecker{roleName: requiredRole}

	tests := map[string]struct {
		roleNames []string
		wantErr   error
	}{
		"nil role list": {
			roleNames: nil,
			wantErr:   auth.ErrNoValidRole,
		},
		"empty role list": {
			roleNames: []string{},
			wantErr:   auth.ErrNoValidRole,
		},
		"single matching role": {
			roleNames: []string{requiredRole},
		},
		"single non-matching role": {
			roleNames: []string{"Analyst"},
			wantErr:   errox.NotAuthorized,
		},
		"multiple roles, one matches": {
			roleNames: []string{"Analyst", requiredRole, "Vuln Manager"},
		},
		"multiple roles, none match": {
			roleNames: []string{"Analyst", "Vuln Manager"},
			wantErr:   errox.NotAuthorized,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.ErrorIs(it, checker.checkRole(tt.roleNames), tt.wantErr)
		})
	}
}

func TestWithRole_ReturnsAuthorizer(t *testing.T) {
	assert.NotNil(t, WithRole("Admin"))
}

// ErrNoValidRole wraps errox.NoCredentials — verify the wrapping is preserved.
func TestErrNoValidRole_WrapsNoCredentials(t *testing.T) {
	assert.ErrorIs(t, auth.ErrNoValidRole, errox.NoCredentials)
}
