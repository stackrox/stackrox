package processor

import (
	"context"

	"github.com/stackrox/stackrox/central/notifiers"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/integrationhealth"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Processor is the interface for processing benchmarks, notifiers, and policies.
//go:generate mockgen-wrapper
type Processor interface {
	ProcessAlert(ctx context.Context, alert *storage.Alert)
	ProcessAuditMessage(ctx context.Context, msg *v1.Audit_Message)

	HasNotifiers() bool
	HasEnabledAuditNotifiers() bool

	UpdateNotifier(ctx context.Context, notifier notifiers.Notifier)
	RemoveNotifier(ctx context.Context, id string)
	GetNotifier(ctx context.Context, id string) notifiers.Notifier
	UpdateNotifierHealthStatus(notifier notifiers.Notifier, healthStatus storage.IntegrationHealth_Status, errMessage string)
}

// New returns a new Processor
func New(ns NotifierSet, reporter integrationhealth.Reporter) Processor {
	return &processorImpl{
		ns:       ns,
		reporter: reporter,
	}
}
