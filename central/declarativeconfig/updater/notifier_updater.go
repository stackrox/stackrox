package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/set"
)

type notifierUpdater struct {
	notifierDS    notifierDataStore.DataStore
	policyCleaner policycleaner.PolicyCleaner
	processor     notifier.Processor
	healthDS      declarativeConfigHealth.DataStore
	reporter      integrationhealth.Reporter
	nameExtractor types.NameExtractor
	idExtractor   types.IDExtractor
}

var _ ResourceUpdater = (*notifierUpdater)(nil)

func newNotifierUpdater(notifierDS notifierDataStore.DataStore, policyCleaner policycleaner.PolicyCleaner,
	processor notifier.Processor, healthDS declarativeConfigHealth.DataStore,
	reporter integrationhealth.Reporter) ResourceUpdater {
	return &notifierUpdater{
		notifierDS:    notifierDS,
		policyCleaner: policyCleaner,
		processor:     processor,
		healthDS:      healthDS,
		reporter:      reporter,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
	}
}

func (u *notifierUpdater) Upsert(ctx context.Context, m proto.Message) error {
	notifierProto, ok := m.(*storage.Notifier)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to role updater: %T", notifierProto)
	}
	_, err := u.notifierDS.UpsertNotifier(ctx, notifierProto)
	if err != nil {
		return err
	}
	notifier, err := notifiers.CreateNotifier(notifierProto)
	if err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}
	u.processor.UpdateNotifier(ctx, notifier)

	return u.reporter.Register(notifierProto.GetId(), notifierProto.GetName(), storage.IntegrationHealth_NOTIFIER)
}

func (u *notifierUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	notifiersToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	notifiers, err := u.notifierDS.GetNotifiersFiltered(ctx, func(n *storage.Notifier) bool {
		return declarativeconfig.IsDeclarativeOrigin(n) &&
			!notifiersToSkip.Contains(n.GetId())
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative notifiers")
	}

	var notifierDeletionErr *multierror.Error
	var notifierIDs []string
	for _, n := range notifiers {
		if err := u.policyCleaner.DeleteNotifierFromPolicies(n.GetId()); err != nil {
			notifierDeletionErr, notifierIDs = u.processDeletionError(ctx, notifierDeletionErr, errors.Wrap(err, "deleting notifier from policies"), notifierIDs, n)
			continue
		}

		// In case of temporary issues with database(for example, connectivity issues)
		// it is possible that notifier is already deleted from datastore
		// while integration health isn't.
		if err := u.notifierDS.RemoveNotifier(ctx, n.GetId()); err != nil && !errors.Is(err, errox.NotFound) {
			err := errors.Wrap(err, "deleting notifier from database")
			notifierDeletionErr, notifierIDs = u.processDeletionError(ctx, notifierDeletionErr, err, notifierIDs, n)
			continue
		}

		u.processor.RemoveNotifier(ctx, n.GetId())
		if err := u.reporter.RemoveIntegrationHealth(n.GetId()); err != nil {
			err := errors.Wrap(err, "deleting notifier's integration health")
			notifierDeletionErr, notifierIDs = u.processDeletionError(ctx, notifierDeletionErr, err, notifierIDs, n)
		}
	}
	return notifierIDs, notifierDeletionErr.ErrorOrNil()
}

func (u *notifierUpdater) processDeletionError(ctx context.Context, notifierDeletionErr *multierror.Error, err error,
	notifierIDs []string, notifier *storage.Notifier) (*multierror.Error, []string) {
	notifierDeletionErr = multierror.Append(notifierDeletionErr, err)
	notifierIDs = append(notifierIDs, notifier.GetId())

	if err := u.healthDS.UpdateStatusForDeclarativeConfig(ctx, u.idExtractor(notifier), err); err != nil {
		log.Errorf("Failed to update the declarative config health status %q: %v", notifier.GetId(), err)
	}
	return notifierDeletionErr, notifierIDs
}
