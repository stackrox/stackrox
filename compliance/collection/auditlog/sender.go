package auditlog

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// auditLogSender provides functionality to send audit events to Sensor.
type auditLogSender interface {
	// Send sends the specified event to Sensor. The events may be buffered
	Send(ctx context.Context, event *auditEvent) error
}

// newAuditLogSender returns a new instance of AuditLogSender
func newAuditLogSender(client sensor.ComplianceService_CommunicateClient, nodeName string, clusterID string) auditLogSender {
	return &auditLogSenderImpl{
		client:    client,
		nodeName:  nodeName,
		clusterID: clusterID,
	}
}
