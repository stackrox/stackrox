package notifiers

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// AuditNotifier is the notifier for audit logs
//
//go:generate mockgen-wrapper AuditNotifier
type AuditNotifier interface {
	Notifier
	// SendAuditMessage sends an audit message
	SendAuditMessage(ctx context.Context, msg *v1.Audit_Message) error
	AuditLoggingEnabled() bool
}
