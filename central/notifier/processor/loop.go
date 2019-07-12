package processor

import (
	"time"

	"github.com/stackrox/rox/central/notifiers"
)

// Loop retries all of the failed alerts for each notifier every hour.
type Loop interface {
	Start()
}

// NewLoop returns a new instance of a Loop.
func NewLoop(pns policyNotifierSet) Loop {
	return &loopImpl{
		pns: pns,
	}
}

type loopImpl struct {
	pns policyNotifierSet
}

func (l *loopImpl) Start() {
	go l.run()
}

func (l *loopImpl) run() {
	ticker := time.NewTicker(retryAlertsEvery)
	for range ticker.C {
		l.retryFailures()
	}
}

func (l *loopImpl) retryFailures() {
	// For every notifier that is tracking it's failure, tell it to retry them all.
	l.pns.forEach(func(notifier notifiers.Notifier) {
		if fr, ok := notifier.(failureRecorder); ok {
			fr.retryFailed()
		}
	})
}
