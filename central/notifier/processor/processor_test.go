package processor

import (
	"testing"

	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

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

func TestProcessorNoEnabledAuditNotifiers(t *testing.T) {
	a := assert.New(t)

	p := New()
	for i := 0; i < 10; i++ {
		p.UpdateNotifier(newFakeAuditNotifier(false))
		p.UpdateNotifier(newFakeNonAuditNotifier())
	}
	a.Equal(false, p.HasEnabledAuditNotifiers())
}

func TestProcessorEnabledAuditNotifiers(t *testing.T) {
	a := assert.New(t)

	p := New()
	nonAudit := newFakeNonAuditNotifier()
	p.UpdateNotifier(nonAudit)
	a.Equal(false, p.HasEnabledAuditNotifiers())
	audit := newFakeAuditNotifier(true)
	p.UpdateNotifier(audit)
	a.Equal(true, p.HasEnabledAuditNotifiers())
	p.RemoveNotifier(nonAudit.ProtoNotifier().GetId())
	a.Equal(true, p.HasEnabledAuditNotifiers())
	p.RemoveNotifier(audit.ProtoNotifier().GetId())
	a.Equal(false, p.HasEnabledAuditNotifiers())
}
