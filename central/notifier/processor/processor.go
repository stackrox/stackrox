package processor

import (
	"github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifications/notifiers"
)

const (
	alertChanSize     = 100
	benchmarkChanSize = 100
)

var (
	log = logging.LoggerForModule()
)

// Processor is the interface for processing benchmarks, notifiers, and policies.
//go:generate mockery -name=Processor
type Processor interface {
	Start()
	ProcessAlert(alert *v1.Alert)
	ProcessBenchmark(schedule *v1.BenchmarkSchedule)

	UpdateNotifier(notifier notifiers.Notifier)
	RemoveNotifier(id string)

	GetIntegratedPolicies(notifierID string) (output []*v1.Policy)
	UpdatePolicy(policy *v1.Policy)
	RemovePolicy(policy *v1.Policy)
}

// New returns a new Processor
func New(storage store.Store) (Processor, error) {
	processor := &processorImpl{
		alertChan:           make(chan *v1.Alert, alertChanSize),
		benchmarkChan:       make(chan *v1.BenchmarkSchedule, benchmarkChanSize),
		notifiers:           make(map[string]notifiers.Notifier),
		notifiersToPolicies: make(map[string]map[string]*v1.Policy),
		storage:             storage,
	}
	err := processor.initializeNotifiers()
	return processor, err
}
