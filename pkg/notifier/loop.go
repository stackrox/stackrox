package notifier

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/notifiers"
)

// Loop retries all failed alerts for each notifier every hour.
type Loop interface {
	Start(context.Context)
	TestRetryFailures(ctx context.Context, t *testing.T)
}

// NewLoop returns a new instance of a Loop.
func NewLoop(ns Set, retryAlertsEvery time.Duration) Loop {
	return &loopImpl{
		retryAlertsEvery: retryAlertsEvery,
		ns:               ns,
	}
}

type loopImpl struct {
	retryAlertsEvery time.Duration
	ns               Set
}

func (l *loopImpl) Start(ctx context.Context) {
	go l.run(ctx)
}

func (l *loopImpl) run(ctx context.Context) {
	ticker := time.NewTicker(l.retryAlertsEvery)
	for range ticker.C {
		l.retryFailures(ctx)
	}
}

func (l *loopImpl) retryFailures(ctx context.Context) {
	// For every notifier that is tracking its failure, tell it to retry them all.
	l.ns.ForEach(ctx, func(ctx context.Context, notifier notifiers.Notifier, failures AlertSet) {
		for _, alert := range failures.GetAll() {
			err := TryToAlert(ctx, notifier, alert)
			if err == nil {
				failures.Remove(alert.GetId())
			}
		}
	})
}

func (l *loopImpl) TestRetryFailures(ctx context.Context, t *testing.T) {
	if t == nil {
		return
	}
	l.retryFailures(ctx)
}
