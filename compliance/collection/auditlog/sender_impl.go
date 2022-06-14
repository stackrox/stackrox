package auditlog

import (
	"context"

	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
)

type auditLogSenderImpl struct {
	client    sensor.ComplianceService_CommunicateClient
	nodeName  string
	clusterID string
}

func (s *auditLogSenderImpl) Send(ctx context.Context, event *auditEvent) error {
	k8sEvent := event.ToKubernetesEvent(s.clusterID)

	return s.client.Send(&sensor.MsgFromCompliance{
		Node: s.nodeName,
		Msg: &sensor.MsgFromCompliance_AuditEvents{
			AuditEvents: &sensor.AuditEvents{
				Events: []*storage.KubernetesEvent{k8sEvent},
			},
		},
	})
}
