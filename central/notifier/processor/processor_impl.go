package processor

import (
	"context"

	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// Replacing with a background context such that outside context cancellation
	// does not affect long running go routines.
	ctxBackground = context.Background()
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
func (p *processorImpl) RemoveNotifier(ctx context.Context, id string) {
	p.ns.RemoveNotifier(ctx, id)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *processorImpl) UpdateNotifier(ctx context.Context, notifier notifiers.Notifier) {
	p.ns.UpsertNotifier(ctx, notifier)
}

// ProcessAlert pushes the alert into a channel to be processed
func (p *processorImpl) ProcessAlert(ctx context.Context, alert *storage.Alert) {
	alertNotifiers := set.NewStringSet(alert.GetPolicy().GetNotifiers()...)
	p.ns.ForEach(ctx, func(ctx context.Context, notifier notifiers.Notifier, failures AlertSet) {
		if alertNotifiers.Contains(notifier.ProtoNotifier().GetId()) {
			go func() {
				err := tryToAlert(ctx, notifier, alert)
				if err != nil {
					failures.Add(alert)
				}
			}()
		}
	})
}

// ProcessAuditMessage sends the audit message with all applicable notifiers.
func (p *processorImpl) ProcessAuditMessage(ctx context.Context, msg *v1.Audit_Message) {
	// TODO: Turn processorImpl into a work queue and introduce func (p *processorImpl) run(context.Context) error.
	// With that, we wouldn't have to fan out n go routines (n = # notifiers in p.ns) and ensure ordering
	// of audit messages.
	p.ns.ForEach(ctx, func(_ context.Context, notifier notifiers.Notifier, _ AlertSet) {
		go tryToSendAudit(ctxBackground, notifier, msg)
	})
}

// Used for testing.
func (p *processorImpl) processAlertSync(ctx context.Context, alert *storage.Alert) {
	alertNotifiers := set.NewStringSet(alert.GetPolicy().GetNotifiers()...)
	p.ns.ForEach(ctx, func(ctx context.Context, notifier notifiers.Notifier, failures AlertSet) {
		if alertNotifiers.Contains(notifier.ProtoNotifier().GetId()) {
			err := tryToAlert(ctx, notifier, alert)
			if err != nil {
				failures.Add(alert)
			}
		}
	})
}
