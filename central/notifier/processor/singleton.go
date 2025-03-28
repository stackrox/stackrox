package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/central/notifier/datastore"
	encConfigDatastore "github.com/stackrox/rox/central/notifier/encconfig/datastore"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// When we fail to notify on an alert, retry every hour for 4 hours, and only retry up to 100 alerts
	retryAlertsEvery = 5 * time.Minute
	retryAlertsFor   = 1 * time.Hour
)

var (
	once sync.Once

	ns   notifier.Set
	loop notifier.Loop
	pr   notifier.Processor
)

func initialize() {
	// Create a context that can access notifiers and namespaces since this is on initialization.
	ctx := declarativeconfig.WithModifyDeclarativeOrImperative(
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Integration, resources.Namespace))))

	// Keep track of the notifiers in use.
	ns = notifier.NewNotifierSet(retryAlertsFor)

	// When alerts are generated, we will want to notify.
	pr = New(ns, reporter.Singleton())

	notifierDatastore := datastore.Singleton()

	if env.EncNotifierCreds.BooleanSetting() {
		var notifiersToUpsert []*storage.Notifier
		cryptoKey, activeIndex, err := notifierUtils.GetActiveNotifierEncryptionKey()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error reading encryption key, notifiers will be unable to send notifications"))
		}
		encConfigDataStore := encConfigDatastore.Singleton()
		encConfig, err := encConfigDataStore.GetConfig()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error getting notifier encryption config"))
		}
		if encConfig == nil {
			// This will be true when secured notifiers feature is enabled for the first time as the config will not exist yet in the db
			encConfig = &storage.NotifierEncConfig{ActiveKeyIndex: 0}
			err = encConfigDataStore.UpsertConfig(encConfig)
			if err != nil {
				utils.Should(errors.Wrap(err, "Error inserting notifier encryption config %d"))
			}
		}
		storedKeyIndex := int(encConfig.GetActiveKeyIndex())

		var oldKey string
		var needsRekey bool
		if activeIndex != storedKeyIndex {
			needsRekey = true
			oldKey, err = notifierUtils.GetNotifierEncryptionKeyAtIndex(storedKeyIndex)
			if err != nil {
				utils.Should(errors.Wrap(err, "Error reading old encryption key, notifiers will be unable to send notifications"))
			}
		}

		err = notifierDatastore.ProcessNotifiers(ctx, func(protoNotifier *storage.Notifier) error {
			secured, err := notifierUtils.IsNotifierSecured(protoNotifier)
			if err != nil {
				utils.Should(errors.Wrapf(err, "Error checking id the notifier %s is secured, notifications to this notifier will fail",
					protoNotifier.GetId()))
				return nil
			}
			if !secured {
				// If notifier is not secured, then we just need to secure it using the active key and continue
				err := notifierUtils.SecureNotifier(protoNotifier, cryptoKey)
				if err != nil {
					// Don't send out error from crypto lib
					utils.Should(fmt.Errorf("error securing notifier %s, notifications to this notifier will fail", protoNotifier.GetId()))
					return nil
				}
				notifiersToUpsert = append(notifiersToUpsert, protoNotifier)
				return nil
			}

			if needsRekey {
				// If a notifier is already secured and needsRekey = true i.e (storedKeyIndex != activeKeyIndex)
				// then we need to decrypt using old key and encrypt using the active key
				err = notifierUtils.RekeyNotifier(protoNotifier, oldKey, cryptoKey)
				if err != nil {
					utils.Should(fmt.Errorf("error rekeying notifier %s, notifications to this notifier will fail", protoNotifier.GetId()))
					return nil
				}
				notifiersToUpsert = append(notifiersToUpsert, protoNotifier)
			}
			return nil
		})
		if err != nil {
			log.Panicf("unable to fetch notifiers: %v", err)
		}

		err = notifierDatastore.UpsertManyNotifiers(ctx, notifiersToUpsert)
		if err != nil {
			utils.Should(errors.Wrap(err, "Error upserting secured notifiers, several notifiers will be unable to send notifications"))
		}
		if needsRekey {
			// If we did a rekey, then update the stored key index
			encConfig = &storage.NotifierEncConfig{ActiveKeyIndex: int32(activeIndex)}
			err = encConfigDataStore.UpsertConfig(encConfig)
			if err != nil {
				utils.Should(errors.Wrapf(err, "Error updating notifier encryption config's stored key index to %d", activeIndex))
			}
		}
	}

	// Create actionable notifiers from the loaded protos.
	err := notifierDatastore.ProcessNotifiers(ctx, func(protoNotifier *storage.Notifier) error {
		notifier, err := notifiers.CreateNotifier(protoNotifier)
		if err != nil {
			utils.Should(errors.Wrapf(err, "error creating notifier with %v (%v) and type %v", protoNotifier.GetId(), protoNotifier.GetName(), protoNotifier.GetType()))
			return nil
		}
		pr.UpdateNotifier(ctx, notifier)
		return nil
	})
	if err != nil {
		log.Panicf("unable to fetch notifiers: %v", err)
	}

	// When alerts have failed, we will want to retry the notifications.
	loop = notifier.NewLoop(ns, retryAlertsEvery)
	loop.Start(ctx)
}

// Singleton provides the interface for processing notifications.
func Singleton() notifier.Processor {
	once.Do(initialize)
	return pr
}
