package auditlog

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	// Default location of where the audit log can be found on Compliance
	defaultLogPath = "/host/var/log/kube-apiserver/audit.log"
)

// Reader provides functionality to read, parse and send audit log events to Sensor.
type Reader interface {
	// StartReader will start the audit log reader process which will continuously read and send events until stopped.
	// Returns true if the reader can be started (log exists and can be read). Log file missing is not considered an error.
	StartReader(ctx context.Context) (bool, error)
	// StopReader will stop the reader if it's started.
	StopReader()
}

// NewReader returns a new instance of Reader
func NewReader(client sensor.ComplianceService_CommunicateClient, nodeName string, clusterID string, startState *storage.AuditLogFileState) Reader {
	return &auditLogReaderImpl{
		logPath:    defaultLogPath,
		stopper:    concurrency.NewStopper(),
		sender:     newAuditLogSender(client, nodeName, clusterID),
		startState: startState,
	}
}
