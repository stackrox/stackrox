package compliance

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type mockServer struct {
	sensor.ComplianceService_CommunicateServer
	sentList []*sensor.MsgToCompliance
}

func (c *mockServer) Send(msg *sensor.MsgToCompliance) error {
	c.sentList = append(c.sentList, msg)
	return nil
}

func (c *mockServer) Recv() (*sensor.MsgFromCompliance, error) {
	return nil, nil
}

func TestAuditLogCollectionManager(t *testing.T) {
	suite.Run(t, new(AuditLogCollectionManagerTestSuite))
}

type AuditLogCollectionManagerTestSuite struct {
	suite.Suite

	envIsolator *envisolator.EnvIsolator
}

func (s *AuditLogCollectionManagerTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.K8sAuditLogDetection.EnvVar(), "true")

	// Some test runs (e.g. go-unit-tests-release will force the flag to be disabled, so skip these tests in those cases)
	if !features.K8sAuditLogDetection.Enabled() {
		s.T().Skipf("%s feature flag not enabled, skipping...", features.K8sAuditLogDetection.Name())
	}
}

func (s *AuditLogCollectionManagerTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *AuditLogCollectionManagerTestSuite) getFakeServersAndStates() (map[string]sensor.ComplianceService_CommunicateServer, map[string]*storage.AuditLogFileState) {
	servers := map[string]sensor.ComplianceService_CommunicateServer{
		"node-a": &mockServer{
			sentList: make([]*sensor.MsgToCompliance, 0),
		},
		"node-b": &mockServer{
			sentList: make([]*sensor.MsgToCompliance, 0),
		},
	}

	fileStates := map[string]*storage.AuditLogFileState{
		"node-a": {
			CollectLogsSince: types.TimestampNow(),
			LastAuditId:      "last-audit-id",
		},
	}
	return servers, fileStates
}

func (s *AuditLogCollectionManagerTestSuite) getClusterID() string {
	return "FAKECLUSTERID"
}

func (s *AuditLogCollectionManagerTestSuite) TestEnableCollectionEnablesOnAllAvailableNodes() {
	servers, _ := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 false,
	}

	manager.EnableCollection()

	s.True(manager.enabled, "Collection should be in an enabled state if EnableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 1, "Server for node %s should have gotten exactly one message sent", node)

		startReq := sentMsgs[0].GetAuditLogCollectionRequest().GetStartReq()
		s.NotNil(startReq, "The message sent should have been a start message")
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestEnableCollectionSendsFileStateIfAvailable() {
	servers, fileStates := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		fileStates:              fileStates,
		enabled:                 false,
	}

	manager.EnableCollection()

	s.Equal(fileStates["node-a"],
		servers["node-a"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())

	s.Nil(servers["node-b"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())
}

func (s *AuditLogCollectionManagerTestSuite) TestEnabledDoesntSendMessageIfAlreadyEnabled() {
	servers, _ := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 true,
	}

	manager.EnableCollection()

	s.True(manager.enabled, "Collection should be in an enabled state if EnableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 0, "No message should have been sent because it was already enabled", node)
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestDisableCollectionDisablesOnAllAvailableNodes() {
	servers, _ := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 true,
	}

	manager.DisableCollection()

	s.False(manager.enabled, "Collection should be in a disabled state if DisableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 1, "Server for node %s should have gotten exactly one message sent", node)

		startReq := sentMsgs[0].GetAuditLogCollectionRequest().GetStopReq()
		s.NotNil(startReq, "The message sent should have been a stop message")
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestDisableDoesntSendMessageIfAlreadyDisabled() {
	servers, _ := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 false,
	}

	manager.DisableCollection()

	s.False(manager.enabled, "Collection should be in a disabled state if DisableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 0, "No message should have been sent because it was already disabled", node)
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdateAuditLogFileStateSendsFileStateToAllAvailableNodes() {
	servers, fileStates := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 true,
	}

	manager.UpdateAuditLogFileState(fileStates)

	s.Equal(fileStates, manager.fileStates)

	s.Equal(fileStates["node-a"],
		servers["node-a"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())

	// Explicitly checking that if we got a nil state we send that down
	s.Nil(servers["node-b"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdateAuditLogFileStateDoesNotSendIfDisabled() {
	servers, fileStates := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 false,
	}

	manager.UpdateAuditLogFileState(fileStates)

	s.Equal(fileStates, manager.fileStates, "Even if disabled the state change should be recorded")

	for _, server := range servers {
		s.Len(server.(*mockServer).sentList, 0, "No start message should have been sent because collection is disabled")
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestAddNodeSendsStartIfEnabled() {
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: make(map[string]sensor.ComplianceService_CommunicateServer),
		enabled:                 true,
	}

	server := &mockServer{
		sentList: make([]*sensor.MsgToCompliance, 0),
	}

	manager.AddEligibleComplianceNode("new-node", server)

	s.Len(manager.eligibleComplianceNodes, 1, "The new node should have been added")
	s.Len(server.sentList, 1, "Server for the new node should have gotten exactly one message sent")

	startReq := server.sentList[0].GetAuditLogCollectionRequest().GetStartReq()
	s.NotNil(startReq, "The message sent should have been a start message")
}

func (s *AuditLogCollectionManagerTestSuite) TestAddNodeDoesNotSendIfDisabled() {
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: make(map[string]sensor.ComplianceService_CommunicateServer),
		enabled:                 false,
	}

	server := &mockServer{
		sentList: make([]*sensor.MsgToCompliance, 0),
	}

	manager.AddEligibleComplianceNode("new-node", server)

	s.Len(manager.eligibleComplianceNodes, 1, "The new node should have been added")
	s.Len(server.sentList, 0, "No start message should have been sent because collection is disabled")
}

func (s *AuditLogCollectionManagerTestSuite) TestRemoveNodeRemovesNodeFromList() {
	servers, _ := s.getFakeServersAndStates()
	manager := &AuditLogCollectionManager{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		enabled:                 false,
	}

	manager.RemoveEligibleComplianceNode("node-b")

	s.Nil(manager.eligibleComplianceNodes["node-b"], "The removed node should no longer be present")
}
