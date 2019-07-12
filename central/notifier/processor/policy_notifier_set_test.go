package processor

import (
	"testing"

	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProcessorNoEnabledAuditNotifiers(t *testing.T) {
	a := assert.New(t)

	pns := newPolicyNotifierSet()
	for i := 0; i < 10; i++ {
		pns.upsertNotifier(newFakeAuditNotifier(false))
		pns.upsertNotifier(newFakeNonAuditNotifier())
	}
	a.Equal(false, pns.hasEnabledAuditNotifiers())
}

func TestProcessorEnabledAuditNotifiers(t *testing.T) {
	a := assert.New(t)

	pns := newPolicyNotifierSet()
	nonAudit := newFakeNonAuditNotifier()
	pns.upsertNotifier(nonAudit)
	a.Equal(false, pns.hasEnabledAuditNotifiers())
	audit := newFakeAuditNotifier(true)
	pns.upsertNotifier(audit)
	a.Equal(true, pns.hasEnabledAuditNotifiers())
	pns.removeNotifier(nonAudit.ProtoNotifier().GetId())
	a.Equal(true, pns.hasEnabledAuditNotifiers())
	pns.removeNotifier(audit.ProtoNotifier().GetId())
	a.Equal(false, pns.hasEnabledAuditNotifiers())
}

// Fake objects for testing.
////////////////////////////

type fakeNonAuditNotifier struct {
	id string
}

func (f fakeNonAuditNotifier) ProtoNotifier() *storage.Notifier {
	return &storage.Notifier{Id: f.id}
}

func (fakeNonAuditNotifier) Test() error {
	panic("implement me")
}

func newFakeNonAuditNotifier() notifiers.Notifier {
	return fakeNonAuditNotifier{id: uuid.NewV4().String()}
}

type fakeAuditNotifier struct {
	id                  string
	auditLoggingEnabled bool
}

func (f fakeAuditNotifier) ProtoNotifier() *storage.Notifier {
	return &storage.Notifier{Id: f.id}
}

func (fakeAuditNotifier) Test() error {
	panic("implement me")
}

func (fakeAuditNotifier) SendAuditMessage(msg *v1.Audit_Message) error {
	panic("implement me")
}

func (f fakeAuditNotifier) AuditLoggingEnabled() bool {
	return f.auditLoggingEnabled
}

func newFakeAuditNotifier(auditLoggingEnabled bool) notifiers.AuditNotifier {
	return fakeAuditNotifier{auditLoggingEnabled: auditLoggingEnabled, id: uuid.NewV4().String()}
}
