package processor

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	pr Processor
)

func initialize() {
	// Create a context that can access notifiers since this is on initialization.
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifier)))

	// Load notifiers.
	pr = New()
	protoNotifiers, err := datastore.Singleton().GetNotifiers(ctx, &v1.GetNotifiersRequest{})
	if err != nil {
		log.Panicf("unable to fetch notifiers: %v", err)
	}

	// Create actionable notifiers from the loaded protos.
	for _, protoNotifier := range protoNotifiers {
		notifier, err := notifiers.CreateNotifier(protoNotifier)
		if err != nil {
			log.Panicf("Error creating notifier with %v (%v) and type %v: %v", protoNotifier.GetId(), protoNotifier.GetName(), protoNotifier.GetType(), err)
		}
		pr.UpdateNotifier(notifier)
	}
}

// Singleton provides the interface for processing notifications.
func Singleton() Processor {
	once.Do(initialize)
	return pr
}
