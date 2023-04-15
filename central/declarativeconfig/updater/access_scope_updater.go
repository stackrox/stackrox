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
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/set"
)

type accessScopeUpdater struct {
	roleDS        roleDataStore.DataStore
	reporter      integrationhealth.Reporter
	idExtractor   types.IDExtractor
	nameExtractor types.NameExtractor
}

var _ ResourceUpdater = (*accessScopeUpdater)(nil)

func newAccessScopeUpdater(datastore roleDataStore.DataStore, reporter integrationhealth.Reporter) ResourceUpdater {
	return &accessScopeUpdater{
		roleDS:        datastore,
		reporter:      reporter,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
	}
}

func (u *accessScopeUpdater) Upsert(ctx context.Context, m proto.Message) error {
	accessScope, ok := m.(*storage.SimpleAccessScope)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to access scope updater: %T", accessScope)
	}
	return u.roleDS.UpsertAccessScope(ctx, accessScope)
}

func (u *accessScopeUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	resourcesToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	scopes, err := u.roleDS.GetAccessScopesFiltered(ctx, func(accessScope *storage.SimpleAccessScope) bool {
		return declarativeconfig.IsDeclarativeOrigin(accessScope) &&
			!resourcesToSkip.Contains(accessScope.GetId())
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative access scopes")
	}

	var scopeDeletionErr *multierror.Error
	var scopeIDs []string
	for _, scope := range scopes {
		if err := u.roleDS.RemoveAccessScope(ctx, scope.GetId()); err != nil {
			scopeDeletionErr = multierror.Append(scopeDeletionErr, err)
			scopeIDs = append(scopeIDs, scope.GetId())
			u.reporter.UpdateIntegrationHealthAsync(utils.IntegrationHealthForProtoMessage(scope, "", err,
				u.idExtractor, u.nameExtractor))
			if errors.Is(err, errox.ReferencedByAnotherObject) {
				scope.Traits.Origin = storage.Traits_DECLARATIVE_ORPHANED
				if err = u.roleDS.UpsertAccessScope(ctx, scope); err != nil {
					scopeDeletionErr = multierror.Append(scopeDeletionErr, errors.Wrap(err, "setting origin to orphaned"))
				}
			}
		}
	}
	return scopeIDs, scopeDeletionErr.ErrorOrNil()
}
