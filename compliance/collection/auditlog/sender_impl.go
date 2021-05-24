package auditlog

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
)

type auditLogSenderImpl struct {
	client sensor.ComplianceService_CommunicateClient
}

func (s *auditLogSenderImpl) Send(ctx context.Context, event *auditEvent) error {
	// TODO: Implement
	return nil
}
