package processor

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ns   NotifierSet
	loop Loop
	pr   Processor
)

func initialize() {
	// Create a context that can access notifiers and namespaces since this is on initialization.
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifier, resources.Namespace)))

	// Keep track of the notifiers in use.
	ns = NewNotifierSet()

	// When alerts are generated, we will want to notify.
	pr = New(ns, reporter.Singleton())
	protoNotifiers, err := datastore.Singleton().GetNotifiers(ctx)
	if err != nil {
		log.Panicf("unable to fetch notifiers: %v", err)
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
	loop = NewLoop(ns)
	loop.Start(ctx)
}

// Singleton provides the interface for processing notifications.
func Singleton() Processor {
	once.Do(initialize)
	return pr
}
