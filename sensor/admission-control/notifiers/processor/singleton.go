package processor

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/sync"
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
	ctx := context.Background()

	// Keep track of the notifiers in use.
	ns = notifier.NewNotifierSet(retryAlertsFor)

	// When alerts are generated, we will want to notify.
	pr = New(ns)

	// When alerts have failed, we will want to retry the notifications.
	loop = notifier.NewLoop(ns, retryAlertsEvery)
	loop.Start(ctx)
}

// Singleton provides the interface for processing notifications.
func Singleton() notifier.Processor {
	once.Do(initialize)
	return pr
}
