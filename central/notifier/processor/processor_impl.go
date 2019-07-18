package processor

import (
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// Processor takes in alerts and sends the notifications tied to that alert
type processorImpl struct {
	ns NotifierSet
}

func (p *processorImpl) HasNotifiers() bool {
	return p.ns.HasNotifiers()
}

func (p *processorImpl) HasEnabledAuditNotifiers() bool {
	return p.ns.HasEnabledAuditNotifiers()
}

// RemoveNotifier removes the in memory copy of the specified notifier
func (p *processorImpl) RemoveNotifier(id string) {
	p.ns.RemoveNotifier(id)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *processorImpl) UpdateNotifier(notifier notifiers.Notifier) {
	p.ns.UpsertNotifier(notifier)
}

// ProcessAlert pushes the alert into a channel to be processed
func (p *processorImpl) ProcessAlert(alert *storage.Alert) {
	alertNotifiers := set.NewStringSet(alert.GetPolicy().GetNotifiers()...)
	p.ns.ForEach(func(notifier notifiers.Notifier, failures AlertSet) {
		if alertNotifiers.Contains(notifier.ProtoNotifier().GetId()) {
			go func() {
				err := tryToAlert(notifier, alert)
				if err != nil {
					failures.Add(alert)
				}
			}()
		}
	})
}

// ProcessAuditMessage sends the audit message with all applicable notifiers.
func (p *processorImpl) ProcessAuditMessage(msg *v1.Audit_Message) {
	p.ns.ForEach(func(notifier notifiers.Notifier, _ AlertSet) {
		go tryToSendAudit(notifier, msg)
	})
}

// Used for testing.
func (p *processorImpl) processAlertSync(alert *storage.Alert) {
	alertNotifiers := set.NewStringSet(alert.GetPolicy().GetNotifiers()...)
	p.ns.ForEach(func(notifier notifiers.Notifier, failures AlertSet) {
		if alertNotifiers.Contains(notifier.ProtoNotifier().GetId()) {
			err := tryToAlert(notifier, alert)
			if err != nil {
				failures.Add(alert)
			}
		}
	})
}
