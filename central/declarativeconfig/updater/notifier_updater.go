package updater

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

type notifierUpdater struct {
	notifierDS    notifierDataStore.DataStore
	policyCleaner policycleaner.PolicyCleaner
	processor     notifier.Processor
	healthDS      declarativeConfigHealth.DataStore
	reporter      integrationhealth.Reporter
	nameExtractor types.NameExtractor
	idExtractor   types.IDExtractor
	cryptoKey     string
}

var _ ResourceUpdater = (*notifierUpdater)(nil)

func newNotifierUpdater(notifierDS notifierDataStore.DataStore, policyCleaner policycleaner.PolicyCleaner,
	processor notifier.Processor, healthDS declarativeConfigHealth.DataStore,
	reporter integrationhealth.Reporter) ResourceUpdater {
	var cryptoKey string
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, _, err = notifierUtils.GetActiveNotifierEncryptionKey()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error creating declarative config notifier updater, notifiers will be unable to send notifications"))
		}
	}
	return &notifierUpdater{
		notifierDS:    notifierDS,
		policyCleaner: policyCleaner,
		processor:     processor,
		healthDS:      healthDS,
		reporter:      reporter,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
		cryptoKey:     cryptoKey,
	}
}

func (u *notifierUpdater) Upsert(ctx context.Context, m protocompat.Message) error {
	notifierProto, ok := m.(*storage.Notifier)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to role updater: %T", notifierProto)
	}
	if env.EncNotifierCreds.BooleanSetting() {
		err := notifierUtils.SecureNotifier(notifierProto, u.cryptoKey)
		if err != nil {
			return errors.Errorf("Error securing declarative config notifier %s, notifications to this notifier will fail", notifierProto.GetName())
		}
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

func (u *notifierUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, int, error) {
	notifiersToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	notifiers, err := u.notifierDS.GetNotifiersFiltered(ctx, func(n *storage.Notifier) bool {
		return declarativeconfig.IsDeclarativeOrigin(n) &&
			!notifiersToSkip.Contains(n.GetId())
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "retrieving declarative notifiers")
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
	return notifierIDs, len(notifiers) - len(notifierIDs), notifierDeletionErr.ErrorOrNil()
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
