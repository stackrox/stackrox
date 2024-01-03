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
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration, resources.Namespace)))

	// Keep track of the notifiers in use.
	ns = notifier.NewNotifierSet(retryAlertsFor)

	// When alerts are generated, we will want to notify.
	pr = New(ns, reporter.Singleton())

	notifierDatastore := datastore.Singleton()
	protoNotifiers, err := notifierDatastore.GetNotifiers(ctx)
	if err != nil {
		log.Panicf("unable to fetch notifiers: %v", err)
	}

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
			utils.Should(errors.New("Invalid notifier encryption config, should be non-nil"))
		}

		var oldKey string
		var needsRekey bool
		if activeIndex != int(encConfig.ActiveKeyIndex) {
			needsRekey = true
			oldKey, err = notifierUtils.GetNotifierEncryptionKeyAtIndex(int(encConfig.ActiveKeyIndex))
			if err != nil {
				utils.Should(errors.Wrap(err, "Error reading old encryption key, notifiers will be unable to send notifications"))
			}
		}
		for _, protoNotifier := range protoNotifiers {
			secured, err := notifierUtils.IsNotifierSecured(protoNotifier)
			if err != nil {
				utils.Should(errors.Wrapf(err, "Error checking id the notifier %s is secured, notifications to this notifier may fail",
					protoNotifier.GetId()))
				continue
			}
			if secured {
				if needsRekey {
					err = notifierUtils.RekeyNotifier(protoNotifier, oldKey, cryptoKey)
					if err != nil {
						utils.Should(fmt.Errorf("error rekeying notifier %s, notifications to this notifier will fail", protoNotifier.GetId()))
						continue
					}
					notifiersToUpsert = append(notifiersToUpsert, protoNotifier)
				}
			} else {
				err := notifierUtils.SecureNotifier(protoNotifier, cryptoKey)
				if err != nil {
					// Don't send out error from crypto lib
					utils.Should(fmt.Errorf("Error securing notifier %s, notifications to this notifier will fail", protoNotifier.GetId()))
					continue
				}
				notifiersToUpsert = append(notifiersToUpsert, protoNotifier)
			}
		}
		err = notifierDatastore.UpsertManyNotifiers(ctx, notifiersToUpsert)
		if err != nil {
			utils.Should(errors.Wrap(err, "Error upserting secured notifiers, several ntifiers may be unable to send notifications"))
		}
		if needsRekey {
			encConfig.ActiveKeyIndex = int32(activeIndex)
			err = encConfigDataStore.UpsertConfig(encConfig)
			if err != nil {
				utils.Should(errors.Wrapf(err, "Error updating active key index to %d", activeIndex))
			}
		}
	}

	// Create actionable notifiers from the loaded protos.
	for _, protoNotifier := range protoNotifiers {
		notifier, err := notifiers.CreateNotifier(protoNotifier)
		if err != nil {
			utils.Should(errors.Wrapf(err, "error creating notifier with %v (%v) and type %v", protoNotifier.GetId(), protoNotifier.GetName(), protoNotifier.GetType()))
			continue
		}
		pr.UpdateNotifier(ctx, notifier)
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
