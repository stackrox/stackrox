package store

import (
	"time"

	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	GetTelemetryConfig() (*storage.TelemetryConfiguration, error)
	SetTelemetryConfig(configuration *storage.TelemetryConfiguration) error

	GetNextSendTime() (time.Time, error)
	SetNextSendTime(t time.Time) error
}
