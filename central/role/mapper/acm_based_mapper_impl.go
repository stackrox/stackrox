package mapper

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac/externalrolebroker"
	"github.com/stackrox/rox/pkg/sac/externalrolebroker/acmclient"
	"golang.org/x/oauth2"
	"k8s.io/client-go/rest"
)

type acmBasedMapperImpl struct {
	clientFactory func(ctx context.Context, token string) (externalrolebroker.ACMClient, error)
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
	/*
		userForCtx := &user{
			name:       ud.Attributes["name"][0],
			identifier: ud.Attributes["userid"][0],
			groups:     ud.Attributes["groups"],
		}
	*/
	tokens := ud.Attributes["providerToken"]
	if len(tokens) == 0 || tokens[0] == "" {
		return nil, nil
	}
	log.Info("OAuth Token ", tokens[0])
	var tokenData oauth2.Token
	err := json.Unmarshal([]byte(tokens[0]), &tokenData)
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy("user had no token data to pass on to ACM")
	}
	log.Info("ACM token ", tokenData.AccessToken)
	acmClient, err := rm.clientFactory(ctx, tokenData.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to instantiate ACM client")
	}
	//ctxForACM := request.WithUser(ctx, userForCtx)
	//log.Info("Querying ACM for user", userForCtx)
	roles, err := externalrolebroker.GetResolvedRolesFromACM(ctx, acmClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resolved roles from ACM")
	}
	log.Info(len(roles), " Resolved roles ", roles)
	return roles, nil
}

func defaultACMClientFactory(_ context.Context, token string) (externalrolebroker.ACMClient, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load k8s config")
	}
	cfg.BearerToken = token
	return acmclient.NewACMClientFromConfig(cfg)
}

// NewACMBasedMapper creates a RoleMapper that retrieves roles from ACM UserPermissions.
// It creates an ACM client using in-cluster configuration and uses it to fetch
// user permissions from the ACM clusterview aggregate API.
func NewACMBasedMapper() (permissions.RoleMapper, error) {
	return &acmBasedMapperImpl{
		clientFactory: defaultACMClientFactory,
	}, nil
}

// NewACMBasedMapperWithClient creates a RoleMapper with a custom ACM client.
// This is useful for testing or when you need to provide a custom client configuration.
func NewACMBasedMapperWithClient(clientFactory func(context.Context, string) (externalrolebroker.ACMClient, error)) permissions.RoleMapper {
	return &acmBasedMapperImpl{
		clientFactory: clientFactory,
	}
}
