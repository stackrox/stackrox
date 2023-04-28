package updater

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/proto"
	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// ResourceUpdater handles updates of proto resources within declarative config reconciliation routine.
// Each ResourceUpdater is responsible for updates of specific proto type.
//
//go:generate mockgen-wrapper
type ResourceUpdater interface {
	Upsert(ctx context.Context, m proto.Message) error
	// DeleteResources will delete all proto resources created within declarative config reconciliation, besides
	// the given resource IDs. It will return an error, if errors occurred, and a list of IDs which failed deletion.
	DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error)
}

// DefaultResourceUpdaters return a map from proto type to an ResourceUpdater instance responsible
// for updates for this type.
func DefaultResourceUpdaters(registry authproviders.Registry) map[reflect.Type]ResourceUpdater {
	return map[reflect.Type]ResourceUpdater{
		types.AuthProviderType: newAuthProviderUpdater(authProviderDatastore.Singleton(), registry,
			groupDataStore.Singleton(), reporter.Singleton()),
		types.GroupType:         newGroupUpdater(groupDataStore.Singleton(), reporter.Singleton()),
		types.RoleType:          newRoleUpdater(roleDatastore.Singleton(), groupDataStore.Singleton(), reporter.Singleton()),
		types.PermissionSetType: newPermissionSetUpdater(roleDatastore.Singleton(), reporter.Singleton()),
		types.AccessScopeType:   newAccessScopeUpdater(roleDatastore.Singleton(), reporter.Singleton()),
	}
}
