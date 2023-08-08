package compliance

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/updater"
	"github.com/stretchr/testify/suite"
)

const (
	// Max time to receive health info status. You may want to increase it if you plan to step through the code with debugger.
	updateTimeout = 3 * time.Second
	// How frequently should updater should send updates during tests.
	updateInterval = 1 * time.Millisecond
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
}

func (s *AuditLogCollectionManagerTestSuite) TearDownTest() {
	defer assertNoGoroutineLeaks(s.T())
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

func (s *AuditLogCollectionManagerTestSuite) getManager(
	servers map[string]sensor.ComplianceService_CommunicateServer,
	fileStates map[string]*storage.AuditLogFileState,
) *auditLogCollectionManagerImpl {

	if fileStates == nil {
		fileStates = make(map[string]*storage.AuditLogFileState)
	}

	return &auditLogCollectionManagerImpl{
		clusterIDGetter:         s.getClusterID,
		eligibleComplianceNodes: servers,
		fileStates:              fileStates,
		updateInterval:          updateInterval,
		stopSig:                 concurrency.NewSignal(),
		forceUpdateSig:          concurrency.NewSignal(),
		centralReady:            concurrency.NewSignal(),
		auditEventMsgs:          make(chan *sensor.MsgFromCompliance, 5), // Buffered for the test only
		fileStateUpdates:        make(chan *message.ExpiringMessage),
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestEnableCollectionEnablesOnAllAvailableNodes() {
	servers, _ := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)

	manager.EnableCollection()

	s.True(manager.enabled.Get(), "Collection should be in an enabled state if EnableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 1, "Server for node %s should have gotten exactly one message sent", node)

		startReq := sentMsgs[0].GetAuditLogCollectionRequest().GetStartReq()
		s.NotNil(startReq, "The message sent should have been a start message")
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestEnableCollectionSendsFileStateIfAvailable() {
	servers, fileStates := s.getFakeServersAndStates()
	manager := s.getManager(servers, fileStates)

	manager.EnableCollection()

	s.Equal(fileStates["node-a"],
		servers["node-a"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())

	s.Nil(servers["node-b"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())
}

func (s *AuditLogCollectionManagerTestSuite) TestEnabledDoesntSendMessageIfAlreadyEnabled() {
	servers, _ := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)
	manager.enabled.Set(true) // start out enabled

	manager.EnableCollection()

	s.True(manager.enabled.Get(), "Collection should be in an enabled state if EnableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 0, "No message should have been sent because it was already enabled", node)
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestDisableCollectionDisablesOnAllAvailableNodes() {
	servers, _ := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)
	manager.enabled.Set(true) // start out enabled

	manager.DisableCollection()

	s.False(manager.enabled.Get(), "Collection should be in a disabled state if DisableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 1, "Server for node %s should have gotten exactly one message sent", node)

		startReq := sentMsgs[0].GetAuditLogCollectionRequest().GetStopReq()
		s.NotNil(startReq, "The message sent should have been a stop message")
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestDisableDoesntSendMessageIfAlreadyDisabled() {
	servers, _ := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)

	manager.DisableCollection()

	s.False(manager.enabled.Get(), "Collection should be in a disabled state if DisableCollection() is called")

	for node, server := range servers {
		sentMsgs := server.(*mockServer).sentList
		s.Lenf(sentMsgs, 0, "No message should have been sent because it was already disabled", node)
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdateAuditLogFileStateSendsFileStateToAllAvailableNodes() {
	servers, fileStates := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)
	manager.enabled.Set(true) // start out enabled

	manager.SetAuditLogFileStateFromCentral(fileStates)

	s.Equal(fileStates, manager.fileStates)

	s.Equal(fileStates["node-a"],
		servers["node-a"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())

	// Explicitly checking that if we got a nil state we send that down
	s.Nil(servers["node-b"].(*mockServer).sentList[0].GetAuditLogCollectionRequest().GetStartReq().GetCollectStartState())
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdateAuditLogFileStateDoesNotSendIfDisabled() {
	servers, fileStates := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)

	manager.SetAuditLogFileStateFromCentral(fileStates)

	s.Equal(fileStates, manager.fileStates, "Even if disabled the state change should be recorded")

	for _, server := range servers {
		s.Len(server.(*mockServer).sentList, 0, "No start message should have been sent because collection is disabled")
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestAddNodeSendsStartIfEnabled() {
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	manager.enabled.Set(true) // start out enabled

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
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)

	server := &mockServer{
		sentList: make([]*sensor.MsgToCompliance, 0),
	}

	manager.AddEligibleComplianceNode("new-node", server)

	s.Len(manager.eligibleComplianceNodes, 1, "The new node should have been added")
	s.Len(server.sentList, 0, "No start message should have been sent because collection is disabled")
}

func (s *AuditLogCollectionManagerTestSuite) TestRemoveNodeRemovesNodeFromList() {
	servers, _ := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)

	manager.RemoveEligibleComplianceNode("node-b")

	s.Nil(manager.eligibleComplianceNodes["node-b"], "The removed node should no longer be present")
}

func (s *AuditLogCollectionManagerTestSuite) TestGetLatestFileStatesReturnsCopyOfState() {
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	manager.enabled.Set(true) // start out enabled

	firstState := s.getAuditLogFileState(types.TimestampNow(), "first-id-a")
	// add a state manually
	manager.updateFileState("node-a", firstState.CollectLogsSince, firstState.LastAuditId)

	states := manager.getLatestFileStates()
	s.Equal(
		map[string]*storage.AuditLogFileState{"node-a": firstState},
		states,
	)

	// Update the state and add a new node
	secondState := s.getAuditLogFileState(types.TimestampNow(), "second-id-a")
	manager.updateFileState("node-a", secondState.CollectLogsSince, secondState.LastAuditId)

	altNodeState := s.getAuditLogFileState(types.TimestampNow(), "first-id-b")
	manager.updateFileState("node-b", altNodeState.CollectLogsSince, altNodeState.LastAuditId)

	// The originally retrieved state should not have changed
	s.Equal(
		map[string]*storage.AuditLogFileState{"node-a": firstState},
		states,
	)

	// But when fetched again, the new states should be shown
	s.Equal(
		map[string]*storage.AuditLogFileState{"node-a": secondState, "node-b": altNodeState},
		manager.getLatestFileStates(),
	)
}

func (s *AuditLogCollectionManagerTestSuite) getAuditLogFileState(time *types.Timestamp, lastID string) *storage.AuditLogFileState {
	return &storage.AuditLogFileState{
		CollectLogsSince: time,
		LastAuditId:      lastID,
	}
}

func (s *AuditLogCollectionManagerTestSuite) getAsProtoTime(now time.Time) *types.Timestamp {
	protoTime, err := types.TimestampProto(now)
	s.NoError(err)
	return protoTime
}

func (s *AuditLogCollectionManagerTestSuite) TestStateSaverSavesFileStates() {
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	manager.enabled.Set(true) // start out enabled

	s.NoError(manager.Start())
	defer manager.Stop(nil)

	// Now pass in a few messages and wait for the state to get updated asynchronously
	expectedFileStates := make(map[string]*storage.AuditLogFileState)
	startTime := time.Now()
	for node := 0; node < 2; node++ {
		for i := 0; i < 2; i++ {
			nodeName := fmt.Sprintf("node-%d", node)
			ts := s.getAsProtoTime(startTime.Add(time.Duration(i*10) * time.Minute))
			msg := s.getMsgFromCompliance(nodeName, ts)
			state := s.getAuditLogFileState(ts, msg.GetAuditEvents().Events[0].Id)
			expectedFileStates[nodeName] = state

			manager.AuditMessagesChan() <- msg
		}
	}

	// One more just to ensure that by the point we get the file state, all message before this one has been processed
	manager.AuditMessagesChan() <- s.getMsgFromCompliance("node-X", s.getAsProtoTime(startTime.Add(30*time.Minute)))

	// Wait until the buffer is empty, at which point we know all messages were consumed
	for len(manager.auditEventMsgs) > 0 { // should be safe to do since we are the only reader and because it's a buffered channel
		time.Sleep(5 * time.Second)
	}

	states := manager.getLatestFileStates()
	delete(states, "node-X") // Just in case the test ran fast, and the message added to flush the channel exists, remove it
	s.Equal(expectedFileStates, states)

}

func (s *AuditLogCollectionManagerTestSuite) TestStateSaverDoesNotSaveIfMsgHasNoEvents() {
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	manager.enabled.Set(true) // start out enabled

	s.NoError(manager.Start())
	defer manager.Stop(nil)

	// Now pass in a few messages and wait for the state to get updated asynchronously
	startTime := time.Now()
	for node := 0; node < 2; node++ {
		for i := 0; i < 2; i++ {
			msg := &sensor.MsgFromCompliance{
				Node: fmt.Sprintf("node-%d", node),
				Msg: &sensor.MsgFromCompliance_AuditEvents{
					AuditEvents: &sensor.AuditEvents{Events: []*storage.KubernetesEvent{}},
				},
			}

			manager.AuditMessagesChan() <- msg
		}
	}

	// One more just to ensure that by the point we get the file state, all message before this one has been processed
	manager.AuditMessagesChan() <- s.getMsgFromCompliance("node-X", s.getAsProtoTime(startTime.Add(30*time.Minute)))

	// Wait until the buffer is empty, at which point we know all messages were consumed
	for len(manager.auditEventMsgs) > 0 { // should be safe to do since we are the only reader and because it's a buffered channel
		time.Sleep(5 * time.Second)
	}

	states := manager.getLatestFileStates()
	delete(states, "node-X") // Just in case the test ran fast, and the message added to flush the channel exists, remove it
	s.True(len(states) == 0) // state shouldn't have been updated

}

func (s *AuditLogCollectionManagerTestSuite) getMsgFromCompliance(nodeName string, timestamp *types.Timestamp) *sensor.MsgFromCompliance {
	return &sensor.MsgFromCompliance{
		Node: nodeName,
		Msg: &sensor.MsgFromCompliance_AuditEvents{
			AuditEvents: &sensor.AuditEvents{
				Events: []*storage.KubernetesEvent{
					{
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
					},
				},
			},
		},
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdaterDoesNotSendWhenNoFileStates() {
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)

	err := manager.Start()
	s.Require().NoError(err)
	defer manager.Stop(nil)

	timer := time.NewTimer(updateTimeout + (500 * time.Millisecond)) // wait an extra 1/2 second

	select {
	case <-manager.ResponsesC():
		s.Fail("Received message when updater should not have sent one!")
	case <-timer.C:
		return // successful
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdaterDoesNotSendIfInitStateNotReceivedFromCentral() {
	now := time.Now()
	fileStates := map[string]*storage.AuditLogFileState{
		"node-one": {
			CollectLogsSince: s.getAsProtoTime(now.Add(-10 * time.Minute)),
			LastAuditId:      uuid.NewV4().String(),
		},
	}
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), fileStates)
	manager.receivedInitialStateFromCentral.Set(false)

	err := manager.Start()
	s.Require().NoError(err)
	defer manager.Stop(nil)

	timer := time.NewTimer(updateTimeout + (500 * time.Millisecond)) // wait an extra 1/2 second

	select {
	case <-manager.ResponsesC():
		s.Fail("Received message when updater should not have sent one!")
	case <-timer.C:
		return // successful
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdaterSendsUpdateWithLatestFileStates() {
	now := time.Now()
	expectedStatus := map[string]*storage.AuditLogFileState{
		"node-one": {
			CollectLogsSince: s.getAsProtoTime(now.Add(-10 * time.Minute)),
			LastAuditId:      uuid.NewV4().String(),
		},
		"node-two": {
			CollectLogsSince: s.getAsProtoTime(now.Add(-10 * time.Second)),
			LastAuditId:      uuid.NewV4().String(),
		},
	}

	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), expectedStatus)
	manager.receivedInitialStateFromCentral.Set(true)
	manager.Notify(common.SensorComponentEventCentralReachable)

	err := manager.Start()
	s.Require().NoError(err)
	defer manager.Stop(nil)

	status := s.getUpdaterStatusMsg(manager, 10)
	s.Equal(expectedStatus, status.GetNodeAuditLogFileStates())
}

func (s *AuditLogCollectionManagerTestSuite) TestUpdaterSendsUpdateWhenForced() {
	now := time.Now()
	expectedStatus := map[string]*storage.AuditLogFileState{
		"node-one": {
			CollectLogsSince: s.getAsProtoTime(now.Add(-10 * time.Minute)),
			LastAuditId:      uuid.NewV4().String(),
		},
		"node-two": {
			CollectLogsSince: s.getAsProtoTime(now.Add(-10 * time.Second)),
			LastAuditId:      uuid.NewV4().String(),
		},
	}

	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), expectedStatus)
	// The updater will update a duration that is less than the test timeout, so the update will not be naturally sent until forced
	manager.updateInterval = 1 * time.Minute

	manager.receivedInitialStateFromCentral.Set(true)
	manager.Notify(common.SensorComponentEventCentralReachable)

	err := manager.Start()
	s.Require().NoError(err)
	defer manager.Stop(nil)

	manager.ForceUpdate()

	status := s.getUpdaterStatusMsg(manager, 1)
	s.Equal(expectedStatus, status.GetNodeAuditLogFileStates())
}

func (s *AuditLogCollectionManagerTestSuite) getUpdaterStatusMsg(updater updater.Component, times int) *central.AuditLogStatusInfo {
	timer := time.NewTimer(updateTimeout)

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

// This tests simulates Sensor loosing connection to Central, followed by a reconnect.
// On entering Offline mode, Sensor must not try to send updates to Central.
// As soon as Central comes online, Sensor must run on regular intervals again.
func (s *AuditLogCollectionManagerTestSuite) TestUpdaterSkipsOnOfflineMode() {
	servers, _ := s.getFakeServersAndStates()
	manager := s.getManager(servers, nil)
	manager.auditEventMsgs = make(chan *sensor.MsgFromCompliance)
	defer close(manager.auditEventMsgs)
	manager.receivedInitialStateFromCentral.Set(true)
	s.NoError(manager.Start())

	centralC := manager.ResponsesC()
	complianceC := manager.AuditMessagesChan()

	states := [3]common.SensorComponentEvent{common.SensorComponentEventCentralReachable, common.SensorComponentEventOfflineMode, common.SensorComponentEventCentralReachable}

	for i, state := range states {
		manager.Notify(state)
		complianceC <- s.getMsgFromCompliance(fmt.Sprintf("Node-%d", i), s.getAsProtoTime(time.Now().Add(1*time.Second)))
		select {
		case <-centralC:
			s.T().Logf("Received message on centralC (state: %s)", state)
			if state == common.SensorComponentEventOfflineMode {
				s.Fail("Must not receive messages to central in offline mode")
			}
		case <-time.After(500 * time.Millisecond):
			s.T().Logf("Timeout waiting for a message on centralC (state: %s)", state)
			if state == common.SensorComponentEventCentralReachable {
				s.Fail("CentralC msg didn't arrive within deadline")
				// The message was sent, so we must wait until it finally arrives,
				// otherwise the next iteration may receive it
				s.T().Logf("Timeout happened on %s state, so we must wait for the message", state)
				<-centralC
			}
		}

	}

	manager.Stop(nil)
}
