package processor

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	pr Processor
)

func initialize() {
	pr = New()
	protoNotifiers, err := datastore.Singleton().GetNotifiers(context.TODO(), &v1.GetNotifiersRequest{})
	if err != nil {
		log.Panicf("unable to fetch notifiers: %v", err)
	}

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
