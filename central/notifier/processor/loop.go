package processor

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/notifiers"
)

// Loop retries all of the failed alerts for each notifier every hour.
type Loop interface {
	Start(context.Context)
}

// NewLoop returns a new instance of a Loop.
func NewLoop(ns NotifierSet) Loop {
	return &loopImpl{
		ns: ns,
	}
}

type loopImpl struct {
	ns NotifierSet
}

func (l *loopImpl) Start(ctx context.Context) {
	go l.run(ctx)
}

func (l *loopImpl) run(ctx context.Context) {
	ticker := time.NewTicker(retryAlertsEvery)
	for range ticker.C {
		l.retryFailures(ctx)
	}
}

func (l *loopImpl) retryFailures(ctx context.Context) {
	// For every notifier that is tracking it's failure, tell it to retry them all.
	l.ns.ForEach(ctx, func(ctx context.Context, notifier notifiers.Notifier, failures AlertSet) {
		for _, alert := range failures.GetAll() {
			err := tryToAlert(ctx, notifier, alert)
			if err == nil {
				failures.Remove(alert.GetId())
			}
		}
	})
}
