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
func NewLoop(ns NotifierSet) Loop {
	return &loopImpl{
		ns: ns,
	}
}

type loopImpl struct {
	ns NotifierSet
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
	l.ns.ForEach(func(notifier notifiers.Notifier, failures AlertSet) {
		for _, alert := range failures.GetAll() {
			err := tryToAlert(notifier, alert)
			if err == nil {
				failures.Remove(alert.GetId())
			}
		}
	})
}
