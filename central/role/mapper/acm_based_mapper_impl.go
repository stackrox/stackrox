package mapper

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac/externalrolebroker"
	acmclient "github.com/stackrox/rox/pkg/sac/externalrolebroker/acmclient"
	"k8s.io/apiserver/pkg/endpoints/request"
)

type acmBasedMapperImpl struct {
	acmClient externalrolebroker.ACMClient
}

type user struct {
	name       string
	identifier string
	groups     []string
}

func (u user) GetName() string {
	return u.name
}

func (u user) GetUID() string {
	return u.identifier
}

func (u user) GetGroups() []string {
	return u.groups
}

func (u user) GetExtra() map[string][]string {
	return nil
}

// FromUserDescriptor retrieves roles from ACM UserPermissions.
// It queries the ACM clusterview aggregate API to get user permissions,
// filters them for base Kubernetes resources, and converts them to ACS ResolvedRoles.
func (rm *acmBasedMapperImpl) FromUserDescriptor(ctx context.Context, ud *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	if ud.Attributes == nil || len(ud.Attributes["name"]) <= 0 || len(ud.Attributes["userid"]) <= 0 {
		return nil, errox.InvalidArgs.CausedBy("user had no attribute from which to extract roles")
	}
	userForCtx := &user{
		name:       ud.Attributes["name"][0],
		identifier: ud.Attributes["userid"][0],
		groups:     ud.Attributes["groups"],
	}
	ctxForACM := request.WithUser(ctx, userForCtx)
	log.Info("Querying ACM for user", userForCtx)
	roles, err := externalrolebroker.GetResolvedRolesFromACM(ctxForACM, rm.acmClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resolved roles from ACM")
	}
	log.Info(len(roles), " Resolved roles ", roles)
	return roles, nil
}

// NewACMBasedMapper creates a RoleMapper that retrieves roles from ACM UserPermissions.
// It creates an ACM client using in-cluster configuration and uses it to fetch
// user permissions from the ACM clusterview aggregate API.
func NewACMBasedMapper() (permissions.RoleMapper, error) {
	client, err := acmclient.NewACMClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ACM client")
	}

	return &acmBasedMapperImpl{
		acmClient: client,
	}, nil
}

// NewACMBasedMapperWithClient creates a RoleMapper with a custom ACM client.
// This is useful for testing or when you need to provide a custom client configuration.
func NewACMBasedMapperWithClient(client externalrolebroker.ACMClient) permissions.RoleMapper {
	return &acmBasedMapperImpl{
		acmClient: client,
	}
}
