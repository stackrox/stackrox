package notifier

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/logging/structured"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sync"
)

// Set is a set that coordinates present policies and notifiers.
type Set interface {
	HasNotifiers() bool
	HasEnabledAuditNotifiers() bool

	ForEach(ctx context.Context, f func(context.Context, notifiers.Notifier, AlertSet))

	UpsertNotifier(ctx context.Context, notifier notifiers.Notifier)
	RemoveNotifier(ctx context.Context, id string)
	GetNotifier(ctx context.Context, id string) notifiers.Notifier
	GetNotifiers(ctx context.Context) []notifiers.Notifier
}

// NewNotifierSet returns a new instance of a Set
func NewNotifierSet(retryAlertsFor time.Duration) Set {
	return &notifierSetImpl{
		retryAlertsFor: retryAlertsFor,
		notifiers:      make(map[string]notifiers.Notifier),
		failures:       make(map[string]AlertSet),
	}
}

// Implementation of the notifier set.
//////////////////////////////////////

type notifierSetImpl struct {
	lock sync.RWMutex

	retryAlertsFor time.Duration

	notifiers map[string]notifiers.Notifier
	failures  map[string]AlertSet
}

// HasNotifiers returns if there are any notifiers in the set.
func (p *notifierSetImpl) HasNotifiers() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.notifiers) > 0
}

// HasEnabledAuditNotifiers returns if there are any enabled notifiers in the set.
func (p *notifierSetImpl) HasEnabledAuditNotifiers() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, n := range p.notifiers {
		auditN, ok := n.(notifiers.AuditNotifier)
		if ok && auditN.AuditLoggingEnabled() {
			return true
		}
	}
	return false
}

// ForEach performs a function on each notifier.
func (p *notifierSetImpl) ForEach(ctx context.Context, f func(context.Context, notifiers.Notifier, AlertSet)) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for notifierID, notifier := range p.notifiers {
		f(ctx, notifier, p.failures[notifierID])
	}
}

// UpsertNotifier adds or updates a notifier in the set.
func (p *notifierSetImpl) UpsertNotifier(ctx context.Context, notifier notifiers.Notifier) {
	p.lock.Lock()
	defer p.lock.Unlock()

	notifierID := notifier.ProtoNotifier().GetId()
	if _, exists := p.failures[notifierID]; !exists {
		p.failures[notifierID] = NewAlertSet(p.retryAlertsFor)
	}
	if knownNotifier := p.notifiers[notifierID]; knownNotifier != nil && knownNotifier != notifier {
		if err := knownNotifier.Close(ctx); err != nil {
			log.Error("failed to close notifier instance", structured.Err(err))
		}
	}
	p.notifiers[notifierID] = notifier
}

// RemoveNotifier removes a notifier from the set.
func (p *notifierSetImpl) RemoveNotifier(ctx context.Context, id string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if notifier := p.notifiers[id]; notifier != nil {
		if err := notifier.Close(ctx); err != nil {
			log.Error("failed to close notifier instance", structured.Err(err))
		}
	}

	delete(p.notifiers, id)
	delete(p.failures, id)
}

// GetNotifier gets a notifier from the set.
func (p *notifierSetImpl) GetNotifier(_ context.Context, id string) notifiers.Notifier {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.notifiers[id]
}

// GetNotifiers gets notifiers from the set.
func (p *notifierSetImpl) GetNotifiers(_ context.Context) []notifiers.Notifier {
	p.lock.Lock()
	defer p.lock.Unlock()

	var notifiers []notifiers.Notifier
	for _, notifier := range p.notifiers {
		notifiers = append(notifiers, notifier)
	}
	return notifiers
}
