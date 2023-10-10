package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/central/notifier/datastore"
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

	cryptoKey := ""
	if env.EncNotifierCreds.BooleanSetting() {
		var err error
		cryptoKey, err = notifierUtils.GetNotifierSecretEncryptionKey()
		if err != nil {
			utils.CrashOnError(err)
		}
	}

	protoNotifiers, err := datastore.Singleton().GetNotifiers(ctx)
	if err != nil {
		log.Panicf("unable to fetch notifiers: %v", err)
	}

	// Create actionable notifiers from the loaded protos.
	for _, protoNotifier := range protoNotifiers {
		// Check if the notifier creds are need to be encrypted
		if protoNotifier.GetNotifierSecret() == "" {
			err := notifierUtils.SecureNotifier(protoNotifier, cryptoKey)
			if err != nil {
				// Don't send out error from crypto lib
				utils.CrashOnError(fmt.Errorf("Error securing notifier %s", protoNotifier.GetId()))
			}
			_, err = datastore.Singleton().UpsertNotifier(ctx, protoNotifier)
			if err != nil {
				utils.Should(errors.Wrapf(err, "error upserting secured notifier with %v (%v) and type %v", protoNotifier.GetId(), protoNotifier.GetName(), protoNotifier.GetType()))
				continue
			}
		}

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
