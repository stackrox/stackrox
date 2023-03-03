package updater

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/proto"
	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// ResourceUpdater handles updates of proto resources within declarative config reconciliation routine.
// Each ResourceUpdater is responsible for updates of specific proto type.
// TODO(ROX-14694): Extend interface with methods necessary for resource deletion.
//
//go:generate mockgen-wrapper
type ResourceUpdater interface {
	Upsert(ctx context.Context, m proto.Message) error
}

// DefaultResourceUpdaters return a map from proto type to an ResourceUpdater instance responsible
// for updates for this type.
func DefaultResourceUpdaters(registry authproviders.Registry) map[reflect.Type]ResourceUpdater {
	return map[reflect.Type]ResourceUpdater{
		types.AuthProviderType:  newAuthProviderUpdater(authProviderDatastore.Singleton(), registry),
		types.GroupType:         newGroupUpdater(groupDataStore.Singleton()),
		types.RoleType:          newRoleUpdater(roleDatastore.Singleton()),
		types.PermissionSetType: newPermissionSetUpdater(roleDatastore.Singleton()),
		types.AccessScopeType:   newAccessScopeUpdater(roleDatastore.Singleton()),
	}
}
