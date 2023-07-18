package updater

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/proto"
	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	authProviderRegistry "github.com/stackrox/rox/central/authprovider/registry"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
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
func DefaultResourceUpdaters() map[reflect.Type]ResourceUpdater {
	return map[reflect.Type]ResourceUpdater{
		types.AuthProviderType: newAuthProviderUpdater(authProviderDatastore.Singleton(), authProviderRegistry.Singleton(),
			groupDataStore.Singleton(), declarativeConfigHealth.Singleton()),
		types.GroupType:         newGroupUpdater(groupDataStore.Singleton(), declarativeConfigHealth.Singleton()),
		types.RoleType:          newRoleUpdater(roleDatastore.Singleton(), groupDataStore.Singleton(), declarativeConfigHealth.Singleton()),
		types.PermissionSetType: newPermissionSetUpdater(roleDatastore.Singleton(), declarativeConfigHealth.Singleton()),
		types.AccessScopeType:   newAccessScopeUpdater(roleDatastore.Singleton(), declarativeConfigHealth.Singleton()),
		types.NotifierType: newNotifierUpdater(notifierDataStore.Singleton(), policycleaner.Singleton(),
			notifierProcessor.Singleton(), declarativeConfigHealth.Singleton(), reporter.Singleton()),
	}
}
