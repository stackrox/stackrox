package processor

import (
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Processor is the interface for processing benchmarks, notifiers, and policies.
//go:generate mockgen-wrapper Processor
type Processor interface {
	ProcessAlert(alert *storage.Alert)
	ProcessAuditMessage(msg *v1.Audit_Message)

	HasNotifiers() bool
	HasEnabledAuditNotifiers() bool

	UpdateNotifier(notifier notifiers.Notifier)
	RemoveNotifier(id string)
}

// New returns a new Processor
func New(ns NotifierSet) Processor {
	return &processorImpl{
		ns: ns,
	}
}
