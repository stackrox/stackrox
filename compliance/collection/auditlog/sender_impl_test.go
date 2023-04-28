package auditlog

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type mockClient struct {
	grpc.ClientStream
	sendList []*sensor.MsgFromCompliance
}

func (c *mockClient) Send(msg *sensor.MsgFromCompliance) error {
	c.sendList = append(c.sendList, msg)
	return nil
}

func (c *mockClient) Recv() (*sensor.MsgToCompliance, error) {
	return nil, nil
}

type failingClient struct {
	grpc.ClientStream
}

func (c *failingClient) Send(_ *sensor.MsgFromCompliance) error {
	return errors.New("test fail")
}

func (c *failingClient) Recv() (*sensor.MsgToCompliance, error) {
	return nil, nil
}

func TestComplianceAuditLogSender(t *testing.T) {
	suite.Run(t, new(ComplianceAuditLogSenderTestSuite))
}

type ComplianceAuditLogSenderTestSuite struct {
	suite.Suite
}

func (s *ComplianceAuditLogSenderTestSuite) TestSendCorrectlySendsEvents() {
	client, sender := s.getMocks()

	mockEvent := s.fakeAuditEvent("get", "secrets", "central-mtls", "stackrox")
	err := sender.Send(context.Background(), &mockEvent)
	s.NoError(err)

	s.Len(client.sendList, 1)
	s.validateSentMessage(client.sendList[0], mockEvent)
}

func (s *ComplianceAuditLogSenderTestSuite) TestSendSendsInOrder() {
	client, sender := s.getMocks()

	for i := 0; i < 5; i++ {
		mockEvent := s.fakeAuditEvent("get", "configmaps", fmt.Sprintf("map-%d", i), "stackrox")
		err := sender.Send(context.Background(), &mockEvent)
		s.NoError(err)

		s.Len(client.sendList, i+1)
		s.validateSentMessage(client.sendList[i], mockEvent)
	}
}

func (s *ComplianceAuditLogSenderTestSuite) TestSendReturnsErrorIfClientFails() {
	client := &failingClient{}
	sender := &auditLogSenderImpl{
		client:    client,
		nodeName:  "fakeNodeName",
		clusterID: "test-cluster",
	}

	mockEvent := s.fakeAuditEvent("get", "secrets", "central-mtls", "stackrox")
	s.Error(sender.Send(context.Background(), &mockEvent))
}

func (s *ComplianceAuditLogSenderTestSuite) TestSendCorrectSendsImpersonatedEvent() {
	client, sender := s.getMocks()

	event := s.fakeAuditEvent("get", "secrets", "some-name",
		"ns")
	event.ImpersonatedUser = &userRef{
		Username: "system:serviceaccount:stackrox:central",
		UID:      "",
		Groups:   []string{"system:serviceaccounts", "system:serviceaccounts:stackrox", "system:authenticated"},
	}

	err := sender.Send(context.Background(), &event)
	s.NoError(err)

	s.Len(client.sendList, 1)
	s.validateSentMessage(client.sendList[0], event)
}

func (s *ComplianceAuditLogSenderTestSuite) TestSendCorrectSendsEventWithoutReasonAnnotation() {
	client, sender := s.getMocks()

	event := s.fakeAuditEvent("get", "secrets", "some-name",
		"ns")
	delete(event.Annotations, reasonAnnotationKey)

	err := sender.Send(context.Background(), &event)
	s.NoError(err)

	s.Len(client.sendList, 1)
	msg := client.sendList[0]
	s.validateSentMessage(msg, event)

	s.Empty(msg.GetAuditEvents().Events[0].ResponseStatus.Reason)
}

func (s *ComplianceAuditLogSenderTestSuite) TestSendUsesReceivedTimeIfStageTimeIsInvalid() {
	client, sender := s.getMocks()

	event := s.fakeAuditEvent("get", "secrets", "some-name", "ns")
	event.StageTimestamp = "nan"

	err := sender.Send(context.Background(), &event)
	s.NoError(err)

	s.Len(client.sendList, 1)
	msg := client.sendList[0]
	s.validateSentMessage(msg, event)
}

func (s *ComplianceAuditLogSenderTestSuite) getMocks() (*mockClient, *auditLogSenderImpl) {
	client := &mockClient{
		sendList: []*sensor.MsgFromCompliance{},
	}
	sender := &auditLogSenderImpl{
		client:    client,
		nodeName:  "fakeNodeName",
		clusterID: "test-cluster",
	}
	return client, sender
}

func (s *ComplianceAuditLogSenderTestSuite) validateSentMessage(msg *sensor.MsgFromCompliance, mockEvent auditEvent) {
	s.Equal("fakeNodeName", msg.Node)

	auditEventsMsg := msg.GetAuditEvents()
	s.NotNil(auditEventsMsg)

	events := auditEventsMsg.GetEvents()
	s.Len(events, 1)

	s.Equal(mockEvent.ToKubernetesEvent("test-cluster"), events[0])
}

func (s *ComplianceAuditLogSenderTestSuite) fakeAuditEvent(verb, resourceType, resourceName, namespace string) auditEvent {
	uri := fmt.Sprintf("/api/v1/namespaces/stackrox/%s/%s", resourceType, resourceName)
	event := auditEvent{
		Annotations: map[string]string{
			"authorization.k8s.io/decision": "allow",
			"authorization.k8s.io/reason":   "RBAC: allowed by RoleBinding \"stackrox-central-diagnostics/stackrox\" of Role \"stackrox-central-diagnostics\" to ServiceAccount \"central/stackrox\"",
		},
		APIVersion: "audit.k8s.io/v1",
		AuditID:    uuid.NewV4().String(),
		Kind:       "Event",
		Level:      "Metadata",
		ObjectRef: objectRef{
			APIVersion: "v1",
			Name:       resourceName,
			Namespace:  namespace,
			Resource:   resourceType,
		},
		RequestReceivedTimestamp: "2021-05-06T00:19:49.906385Z",
		RequestURI:               uri,
		ResponseStatus: responseStatusRef{
			Metadata: nil,
			Status:   "",
			Message:  "",
			Code:     200,
		},
		SourceIPs:      []string{"10.0.119.155"},
		Stage:          "ResponseComplete",
		StageTimestamp: "2021-05-06T00:19:49.915375Z",
		User: userRef{
			Username: "cluster-admin",
			UID:      "56d060c4-363a-4d1f-bffc-b146078ccb8e",
			Groups:   []string{"cluster-admins", "system:authenticated:oauth", "system:authenticated"},
		},
		UserAgent: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f",
		Verb:      verb,
	}

	return event
}
