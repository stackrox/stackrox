package auditlog

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/compliance/mocks"
	"github.com/stackrox/rox/sensor/common/updater"
	"github.com/stretchr/testify/suite"
)

const (
	// Max time to receive health info status. You may want to increase it if you plan to step through the code with debugger.
	updateTimeout = 3 * time.Second
	// How frequently should updater should send updates during tests.
	updateInterval = 1 * time.Millisecond
)

func TestUpdater(t *testing.T) {
	suite.Run(t, new(UpdaterTestSuite))
}

type UpdaterTestSuite struct {
	suite.Suite

	auditLogCollectionMgr *mocks.MockAuditLogCollectionManager
	mockCtrl              *gomock.Controller
}

func (s *UpdaterTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.auditLogCollectionMgr = mocks.NewMockAuditLogCollectionManager(s.mockCtrl)
}

func (s *UpdaterTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *UpdaterTestSuite) TestUpdaterDoesNotSendWhenNoFileStates() {
	updater := NewUpdater(updateInterval, s.auditLogCollectionMgr)
	emptyStates := make(map[string]*storage.AuditLogFileState)

	s.auditLogCollectionMgr.EXPECT().GetLatestFileStates().Return(emptyStates).AnyTimes()

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

func (s *UpdaterTestSuite) TestUpdaterSendsUpdateWithLatestFileStates() {
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

	s.auditLogCollectionMgr.EXPECT().GetLatestFileStates().Return(expectedStatus).AnyTimes()

	fileStateUpdater := NewUpdater(updateInterval, s.auditLogCollectionMgr)

	err := fileStateUpdater.Start()
	s.Require().NoError(err)
	defer fileStateUpdater.Stop(nil)

	status := s.getUpdaterStatusMsg(fileStateUpdater, 10)
	s.Equal(expectedStatus, status.GetNodeAuditLogFileStates())
}

func (s *UpdaterTestSuite) TestUpdaterSendsUpdateWhenForced() {
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

	s.auditLogCollectionMgr.EXPECT().GetLatestFileStates().Return(expectedStatus).AnyTimes()

	// The updater will update a duration that is less than the test timeout, so the update will not be naturally sent until forced
	fileStateUpdater := NewUpdater(1*time.Minute, s.auditLogCollectionMgr)

	err := fileStateUpdater.Start()
	s.Require().NoError(err)
	defer fileStateUpdater.Stop(nil)

	fileStateUpdater.ForceUpdate()

	status := s.getUpdaterStatusMsg(fileStateUpdater, 1)
	s.Equal(expectedStatus, status.GetNodeAuditLogFileStates())
}

func (s *UpdaterTestSuite) getUpdaterStatusMsg(updater updater.Component, times int) *central.AuditLogStatusInfo {
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

func (s *UpdaterTestSuite) getAsProtoTime(now time.Time) *types.Timestamp {
	protoTime, err := types.TimestampProto(now)
	s.NoError(err)
	return protoTime
}
