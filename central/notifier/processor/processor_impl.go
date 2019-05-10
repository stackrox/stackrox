package processor

import (
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// Processor takes in alerts and sends the notifications tied to that alert
type processorImpl struct {
	notifiers     map[string]notifiers.Notifier
	notifiersLock sync.RWMutex

	notifiersToPolicies     map[string]map[string]*storage.Policy
	notifiersToPoliciesLock sync.RWMutex
}

func (p *processorImpl) HasNotifiers() bool {
	p.notifiersLock.RLock()
	defer p.notifiersLock.RUnlock()
	return len(p.notifiers) != 0
}

func sendAuditMessage(notifier notifiers.AuditNotifier, msg *v1.Audit_Message) {
	if err := notifier.SendAuditMessage(msg); err != nil {
		protoNotifier := notifier.ProtoNotifier()
		log.Errorf("Unable to send audit msg to %s (%s): %v", protoNotifier.GetName(), protoNotifier.GetType(), err)
	}
}

func sendAlert(notifier notifiers.AlertNotifier, alert *storage.Alert) {
	if err := notifier.AlertNotify(alert); err != nil {
		protoNotifier := notifier.ProtoNotifier()
		log.Errorf("Unable to send %s notification to %s (%s) for alert %s: %v", alert.GetState().String(), protoNotifier.GetName(), protoNotifier.GetType(), alert.GetId(), err)
	}
}

func sendResolvableAlert(notifier notifiers.ResolvableAlertNotifier, alert *storage.Alert) {
	protoNotifier := notifier.ProtoNotifier()
	var err error
	switch alert.GetState() {
	case storage.ViolationState_SNOOZED:
		err = notifier.AckAlert(alert)
	case storage.ViolationState_RESOLVED:
		err = notifier.ResolveAlert(alert)
	}
	if err != nil {
		log.Errorf("Unable to send %s notification to %s (%s) for alert %s: %v", alert.GetState().String(), protoNotifier.GetName(), protoNotifier.GetType(), alert.GetId(), err)
	}
}

// ProcessAlert pushes the alert into a channel to be processed
func (p *processorImpl) ProcessAlert(alert *storage.Alert) {
	p.notifiersLock.RLock()
	defer p.notifiersLock.RUnlock()

	for _, id := range alert.Policy.Notifiers {
		notifier, exists := p.notifiers[id]
		if !exists {
			log.Errorf("Could not send notification to notifier id %s for alert %s because it does not exist", id, alert.GetId())
			continue
		}
		if alert.GetState() == storage.ViolationState_ACTIVE {
			alertNotifier, ok := notifier.(notifiers.AlertNotifier)
			if !ok {
				continue
			}
			go sendAlert(alertNotifier, alert)
		} else {
			alertNotifier, ok := notifier.(notifiers.ResolvableAlertNotifier)
			if !ok {
				continue
			}
			go sendResolvableAlert(alertNotifier, alert)
		}
	}
}

func (p *processorImpl) ProcessAuditMessage(msg *v1.Audit_Message) {
	p.notifiersLock.RLock()
	defer p.notifiersLock.RUnlock()
	for _, n := range p.notifiers {
		auditNotifier, ok := n.(notifiers.AuditNotifier)
		if !ok {
			continue
		}
		go sendAuditMessage(auditNotifier, msg)
	}
}

// RemoveNotifier removes the in memory copy of the specified notifier
func (p *processorImpl) RemoveNotifier(id string) {
	p.notifiersLock.Lock()
	defer p.notifiersLock.Unlock()
	delete(p.notifiers, id)

	p.notifiersToPoliciesLock.Lock()
	defer p.notifiersToPoliciesLock.Unlock()

	delete(p.notifiersToPolicies, id)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *processorImpl) UpdateNotifier(notifier notifiers.Notifier) {
	p.notifiersLock.Lock()
	defer p.notifiersLock.Unlock()
	p.notifiers[notifier.ProtoNotifier().GetId()] = notifier
}

// GetIntegratedPolicies returns a list of policies that use provided notifier.
func (p *processorImpl) GetIntegratedPolicies(notifierID string) (output []*storage.Policy) {
	p.notifiersToPoliciesLock.RLock()
	defer p.notifiersToPoliciesLock.RUnlock()

	if _, ok := p.notifiersToPolicies[notifierID]; !ok {
		return
	}

	output = make([]*storage.Policy, 0, len(p.notifiersToPolicies[notifierID]))
	for _, policy := range p.notifiersToPolicies[notifierID] {
		output = append(output, policy)
	}

	return
}

// UpdatePolicy updates the mapping of notifiers to policies.
func (p *processorImpl) UpdatePolicy(policy *storage.Policy) {
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
			p.notifiersToPolicies[n] = make(map[string]*storage.Policy)
		}

		p.notifiersToPolicies[n][policy.GetId()] = policy
	}
}

// RemovePolicy removes policy from notifiers to policies map.
func (p *processorImpl) RemovePolicy(policy *storage.Policy) {
	p.notifiersToPoliciesLock.Lock()
	defer p.notifiersToPoliciesLock.Unlock()

	for _, n := range policy.GetNotifiers() {
		if p.notifiersToPolicies[n] != nil {
			delete(p.notifiersToPolicies[n], policy.GetId())
		}
	}
}
