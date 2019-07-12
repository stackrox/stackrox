package processor

import (
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Processor takes in alerts and sends the notifications tied to that alert
type processorImpl struct {
	pns policyNotifierSet
}

func (p *processorImpl) HasNotifiers() bool {
	return p.pns.hasNotifiers()
}

func (p *processorImpl) HasEnabledAuditNotifiers() bool {
	return p.pns.hasEnabledAuditNotifiers()
}

// RemoveNotifier removes the in memory copy of the specified notifier
func (p *processorImpl) RemoveNotifier(id string) {
	p.pns.removeNotifier(id)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *processorImpl) UpdateNotifier(notifier notifiers.Notifier) {
	p.pns.upsertNotifier(recordFailures(notifier))
}

// UpdatePolicy updates the mapping of notifiers to policies.
func (p *processorImpl) UpdatePolicy(policy *storage.Policy) {
	p.pns.upsertPolicy(policy)
}

// RemovePolicy removes policy from notifiers to policies map.
func (p *processorImpl) RemovePolicy(policy *storage.Policy) {
	p.pns.removePolicy(policy)
}

// ProcessAlert pushes the alert into a channel to be processed
func (p *processorImpl) ProcessAlert(alert *storage.Alert) {
	p.pns.forEachIntegratedWith(alert.GetPolicy().GetId(), func(notifier notifiers.Notifier) {
		go func() {
			_ = tryToAlert(notifier, alert)
		}()
	})
}

// ProcessAuditMessage sends the audit message with all applicable notifiers.
func (p *processorImpl) ProcessAuditMessage(msg *v1.Audit_Message) {
	p.pns.forEach(func(notifier notifiers.Notifier) {
		go tryToSendAudit(notifier, msg)
	})
}

// Used for testing.
func (p *processorImpl) processAlertSync(alert *storage.Alert) {
	p.pns.forEachIntegratedWith(alert.GetPolicy().GetId(), func(notifier notifiers.Notifier) {
		_ = tryToAlert(notifier, alert)
	})
}
