package service

import (
	"context"

	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/rhsso"
	"github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

type groupCreator func(string, *config.Config) *storage.Group

var (
	log           = logging.LoggerForModule()
	groupCreators = []groupCreator{
		func(id string, _ *config.Config) *storage.Group {
			return &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: id,
					Key:            authproviders.GroupsAttribute,
					Value:          "org_admin", //TODO: make a public constant
				},
				RoleName: role.Admin,
			}
		},
		func(id string, cfg *config.Config) *storage.Group {
			return &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: id,
					Key:            authproviders.UseridAttribute,
					Value:          cfg.RhSso.OwnerUserId,
				},
				RoleName: role.Admin,
			}
		},
		func(id string, _ *config.Config) *storage.Group {
			return &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: id,
				},
				RoleName: role.None,
			}
		},
	}
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.AuthProviderServiceServer
}

func NewWithDefaultProvider(registry authproviders.Registry, groupStore groupDataStore.DataStore) Service {
	service := New(registry, groupStore)

	cfg := config.GetConfig()
	clientSecret := rhsso.LoadRhSsoSecret()

	// TODO: remove
	log.Warn(clientSecret)

	// TODO: limit access
	ctx := sac.WithAllAccess(context.Background())
	provider, err := service.PostAuthProvider(ctx, &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Name:       "Red Hat(SSO)",
			Type:       "oidc",
			UiEndpoint: cfg.RhSso.UiEndpoint,
			Enabled:    true,
			Validated:  true,
			RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
				{
					AttributeKey:   "orgid",
					AttributeValue: cfg.RhSso.OrgId,
				},
			},
			Config: map[string]string{
				"issuer":                       "https://sso.stage.redhat.com/auth/realms/redhat-external", //TODO: depends on stage vs prod env
				"client_id":                    cfg.RhSso.ClientId,
				"client_secret":                clientSecret,
				"mode":                         "post",
				"disable_offline_access_scope": "true",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	for _, creatorFunc := range groupCreators {
		err = groupStore.Upsert(ctx, creatorFunc(provider.GetId(), cfg))
		if err != nil {
			panic(err)
		}
	}
	return service
}

// New returns a new Service instance using the given DataStore.
func New(registry authproviders.Registry, groupStore groupDataStore.DataStore) Service {
	return &serviceImpl{
		registry:   registry,
		groupStore: groupStore,
	}
}
