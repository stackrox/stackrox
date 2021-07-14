package auditlog

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	// Max time to receive health info status. You may want to increase it if you plan to step through the code with debugger.
	updateTimeout = 3 * time.Second
	// How frequently should updater provide health info during tests.
	updateInterval = 1 * time.Millisecond
)

func TestUpdater(t *testing.T) {
	suite.Run(t, new(UpdaterTestSuite))
}

type UpdaterTestSuite struct {
	suite.Suite

	envIsolator *envisolator.EnvIsolator
}

func (s *UpdaterTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *UpdaterTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *UpdaterTestSuite) TestUpdater() {
	s.envIsolator.Setenv(features.K8sAuditLogDetection.EnvVar(), "true")
	if !features.K8sAuditLogDetection.Enabled() {
		s.T().Skipf("%s feature flag not enabled, skipping...", features.K8sAuditLogDetection.Name())
	}

	now := time.Now()
	node1Msg := s.getMsgFromCompliance("node-one", []*storage.KubernetesEvent{
		s.getKubeEvent(s.getAsProtoTime(now.Add(-10 * time.Minute))),
		s.getKubeEvent(s.getAsProtoTime(now.Add(-30 * time.Minute))),
	})
	node2Msg := s.getMsgFromCompliance("node-two", []*storage.KubernetesEvent{
		s.getKubeEvent(s.getAsProtoTime(now.Add(-30 * time.Second))),
	})
	latestNode2Msg := s.getMsgFromCompliance("node-two", []*storage.KubernetesEvent{
		s.getKubeEvent(s.getAsProtoTime(now.Add(-10 * time.Second))),
	})

	auditEventMsgs := make(chan *sensor.MsgFromCompliance, 5)
	auditEventMsgs <- node1Msg
	auditEventMsgs <- node2Msg
	auditEventMsgs <- latestNode2Msg

	expectedStatus := map[string]*storage.AuditLogFileState{
		"node-one": {
			CollectLogsSince: node1Msg.GetAuditEvents().Events[0].Timestamp,
			LastAuditId:      node1Msg.GetAuditEvents().Events[0].Id,
		},
		"node-two": {
			CollectLogsSince: latestNode2Msg.GetAuditEvents().Events[0].Timestamp,
			LastAuditId:      latestNode2Msg.GetAuditEvents().Events[0].Id,
		},
	}

	status := s.getUpdaterStatusMsg(10, auditEventMsgs)

	s.Equal(expectedStatus, status.GetNodeAuditLogFileStates())
}

func (s *UpdaterTestSuite) TestUpdaterDoesNotSendWhenNoFileStates() {
	s.envIsolator.Setenv(features.K8sAuditLogDetection.EnvVar(), "true")
	if !features.K8sAuditLogDetection.Enabled() {
		s.T().Skipf("%s feature flag not enabled, skipping...", features.K8sAuditLogDetection.Name())
	}

	updater := NewUpdater(updateInterval, make(chan *sensor.MsgFromCompliance, 5))

	err := updater.Start()
	s.Require().NoError(err)
	defer updater.Stop(nil)

	timer := time.NewTimer(updateTimeout + (500 * time.Millisecond)) // wait an extra 1/2 second

	select {
	case <-updater.ResponsesC():
		s.Fail("Received message when updater should not have sent one!")
	case <-timer.C:
		return // successful
	}
}

func (s *UpdaterTestSuite) getUpdaterStatusMsg(times int, auditEventMsgs <-chan *sensor.MsgFromCompliance) *central.AuditLogStatusInfo {
	timer := time.NewTimer(updateTimeout)
	updater := NewUpdater(updateInterval, auditEventMsgs)

	err := updater.Start()
	s.Require().NoError(err)
	defer updater.Stop(nil)

	var status *central.AuditLogStatusInfo

	for i := 0; i < times; i++ {
		select {
		case response := <-updater.ResponsesC():
			status = response.Msg.(*central.MsgFromSensor_AuditLogStatusInfo).AuditLogStatusInfo
		case <-timer.C:
			s.Fail("Timed out while waiting for audit log file state update")
		}
	}

	return status
}

func (s *UpdaterTestSuite) getAsProtoTime(now time.Time) *types.Timestamp {
	protoTime, err := types.TimestampProto(now)
	s.NoError(err)
	return protoTime
}

func (s *UpdaterTestSuite) getMsgFromCompliance(nodeName string, events []*storage.KubernetesEvent) *sensor.MsgFromCompliance {
	return &sensor.MsgFromCompliance{
		Node: nodeName,
		Msg: &sensor.MsgFromCompliance_AuditEvents{
			AuditEvents: &sensor.AuditEvents{
				Events: events,
			},
		},
	}
}

func (s *UpdaterTestSuite) getKubeEvent(timestamp *types.Timestamp) *storage.KubernetesEvent {
	return &storage.KubernetesEvent{
		Id: uuid.NewV4().String(),
		Object: &storage.KubernetesEvent_Object{
			Name:      "my-secret",
			Resource:  storage.KubernetesEvent_Object_SECRETS,
			ClusterId: uuid.NewV4().String(),
			Namespace: "my-namespace",
		},
		Timestamp: timestamp,
		ApiVerb:   storage.KubernetesEvent_CREATE,
		User: &storage.KubernetesEvent_User{
			Username: "username",
			Groups:   []string{"groupA", "groupB"},
		},
		SourceIps: []string{"192.168.1.1", "127.0.0.1"},
		UserAgent: "curl",
		ResponseStatus: &storage.KubernetesEvent_ResponseStatus{
			StatusCode: 200,
			Reason:     "cause",
		},
	}
}
