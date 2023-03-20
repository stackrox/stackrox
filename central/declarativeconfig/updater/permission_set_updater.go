package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/utils"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/set"
)

type permissionSetUpdater struct {
	roleDS        roleDataStore.DataStore
	reporter      integrationhealth.Reporter
	idExtractor   types.IDExtractor
	nameExtractor types.NameExtractor
}

var _ ResourceUpdater = (*permissionSetUpdater)(nil)

func newPermissionSetUpdater(datastore roleDataStore.DataStore, reporter integrationhealth.Reporter) ResourceUpdater {
	return &permissionSetUpdater{
		roleDS:        datastore,
		reporter:      reporter,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
	}
}

func (u *permissionSetUpdater) Upsert(ctx context.Context, m proto.Message) error {
	permissionSet, ok := m.(*storage.PermissionSet)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to permission set updater: %T", permissionSet)
	}
	return u.roleDS.UpsertPermissionSet(ctx, permissionSet)
}

func (u *permissionSetUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) error {
	resourcesToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	permissionSets, err := u.roleDS.GetPermissionSetsFiltered(ctx, func(permissionSet *storage.PermissionSet) bool {
		return permissionSet.GetTraits().GetOrigin() == storage.Traits_DECLARATIVE &&
			!resourcesToSkip.Contains(permissionSet.GetId())
	})
	if err != nil {
		return errors.Wrap(err, "retrieving declarative permission sets")
	}

	var permissionSetDeletionErr *multierror.Error
	for _, permissionSet := range permissionSets {
		if err := u.roleDS.RemovePermissionSet(ctx, permissionSet.GetId()); err != nil {
			permissionSetDeletionErr = multierror.Append(permissionSetDeletionErr, err)
			u.reporter.UpdateIntegrationHealthAsync(utils.IntegrationHealthForProtoMessage(permissionSet, "", err,
				u.idExtractor, u.nameExtractor))
		}
	}
	return permissionSetDeletionErr.ErrorOrNil()
}
