package notifications

import (
	"fmt"
	"sync"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
)

const (
	alertChanSize     = 100
	benchmarkChanSize = 100
)

var (
	log = logging.LoggerForModule()
)

// Processor takes in alerts and sends the notifications tied to that alert
type Processor struct {
	alertChan     chan *v1.Alert
	benchmarkChan chan *v1.BenchmarkSchedule
	notifiers     map[string]notifiers.Notifier
	notifiersLock sync.Mutex

	notifiersToPolicies     map[string]map[string]*v1.Policy
	notifiersToPoliciesLock sync.RWMutex

	database db.NotifierStorage
}

// NewNotificationProcessor returns a new Processor
func NewNotificationProcessor(database db.NotifierStorage) (*Processor, error) {
	processor := &Processor{
		alertChan:           make(chan *v1.Alert, alertChanSize),
		benchmarkChan:       make(chan *v1.BenchmarkSchedule, benchmarkChanSize),
		notifiers:           make(map[string]notifiers.Notifier),
		notifiersToPolicies: make(map[string]map[string]*v1.Policy),
		database:            database,
	}
	err := processor.initializeNotifiers()
	return processor, err
}

func (p *Processor) initializeNotifiers() error {
	protoNotifiers, err := p.database.GetNotifiers(&v1.GetNotifiersRequest{})
	if err != nil {
		return err
	}
	for _, protoNotifier := range protoNotifiers {
		notifierCreator, ok := notifiers.Registry[protoNotifier.Type]
		if !ok {
			return fmt.Errorf("Stored notifier type %v does not exist", protoNotifier.Type)
		}
		notifier, err := notifierCreator(protoNotifier)
		if err != nil {
			return fmt.Errorf("Error creating notifier with %v (%v) and type %v: %v", protoNotifier.GetId(), protoNotifier.GetName(), protoNotifier.GetType(), err)
		}
		p.UpdateNotifier(notifier)
	}
	return nil
}

func (p *Processor) notifyAlert(alert *v1.Alert) {
	p.notifiersLock.Lock()
	defer p.notifiersLock.Unlock()
	for _, id := range alert.Policy.Notifiers {
		notifier, exists := p.notifiers[id]
		if !exists {
			log.Errorf("Could not send notification to notifier id %v for alert %v because it does not exist", id, alert.GetId())
			continue
		}
		if err := notifier.AlertNotify(alert); err != nil {
			log.Errorf("Unable to send notification to %v (%v) for alert %v: %v", id, notifier.ProtoNotifier().GetName(), alert.GetId(), err)
		}
	}
}

func (p *Processor) notifyBenchmark(schedule *v1.BenchmarkSchedule) {
	p.notifiersLock.Lock()
	defer p.notifiersLock.Unlock()
	for _, id := range schedule.Notifiers {
		notifier, exists := p.notifiers[id]
		if !exists {
			log.Errorf("Could not send notification to notifier id %v for benchmark %v because it does not exist", id, schedule.GetName())
			continue
		}
		if err := notifier.BenchmarkNotify(schedule); err != nil {
			log.Errorf("Unable to send notification to %v (%v) for benchmark %v: %v", id, notifier.ProtoNotifier().GetName(), schedule.GetName(), err)
		}
	}
}

func (p *Processor) processAlerts() {
	for alert := range p.alertChan {
		p.notifyAlert(alert)
	}
}

func (p *Processor) processBenchmark() {
	for schedule := range p.benchmarkChan {
		p.notifyBenchmark(schedule)
	}
}

// Start begins the notification processor and is blocking
func (p *Processor) Start() {
	go p.processAlerts()
	go p.processBenchmark()
}

// ProcessAlert pushes the alert into a channel to be processed
func (p *Processor) ProcessAlert(alert *v1.Alert) {
	p.alertChan <- alert
}

// ProcessBenchmark pushes the alert into a channel to be processed
func (p *Processor) ProcessBenchmark(schedule *v1.BenchmarkSchedule) {
	p.benchmarkChan <- schedule
}

// RemoveNotifier removes the in memory copy of the specified notifier
func (p *Processor) RemoveNotifier(id string) {
	p.notifiersLock.Lock()
	defer p.notifiersLock.Unlock()
	delete(p.notifiers, id)

	p.notifiersToPoliciesLock.Lock()
	defer p.notifiersToPoliciesLock.Unlock()

	delete(p.notifiersToPolicies, id)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *Processor) UpdateNotifier(notifier notifiers.Notifier) {
	p.notifiersLock.Lock()
	defer p.notifiersLock.Unlock()
	p.notifiers[notifier.ProtoNotifier().GetId()] = notifier
}

// GetIntegratedPolicies returns a list of policies that use provided notifier.
func (p *Processor) GetIntegratedPolicies(notifierID string) (output []*v1.Policy) {
	p.notifiersToPoliciesLock.RLock()
	defer p.notifiersToPoliciesLock.RUnlock()

	if _, ok := p.notifiersToPolicies[notifierID]; !ok {
		return
	}

	output = make([]*v1.Policy, 0, len(p.notifiersToPolicies[notifierID]))
	for _, policy := range p.notifiersToPolicies[notifierID] {
		output = append(output, policy)
	}

	return
}

// UpdatePolicy updates the mapping of notifiers to policies.
func (p *Processor) UpdatePolicy(policy *v1.Policy) {
	p.notifiersToPoliciesLock.Lock()
	defer p.notifiersToPoliciesLock.Unlock()

	for notifier, m := range p.notifiersToPolicies {
		for id := range m {
			if id == policy.GetId() {
				delete(p.notifiersToPolicies[notifier], id)
			}
		}
	}

	for _, n := range policy.GetNotifiers() {
		if p.notifiersToPolicies[n] == nil {
			p.notifiersToPolicies[n] = make(map[string]*v1.Policy)
		}

		p.notifiersToPolicies[n][policy.GetId()] = policy
	}
}

// RemovePolicy removes policy from notifiers to policies map.
func (p *Processor) RemovePolicy(policy *v1.Policy) {
	p.notifiersToPoliciesLock.Lock()
	defer p.notifiersToPoliciesLock.Unlock()

	for _, n := range policy.GetNotifiers() {
		if p.notifiersToPolicies[n] != nil {
			delete(p.notifiersToPolicies[n], policy.GetId())
		}
	}
}
