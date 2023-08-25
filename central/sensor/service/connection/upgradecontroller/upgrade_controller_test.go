package upgradecontroller

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller/stateutils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	fakeClusterID = "FAKE_CLUSTER_ID"

	fakeCurrVersion = "2.5.29.0"

	fakeOldVersion = "2.5.28.0"

	absoluteNoProgressTimeout = 200 * time.Millisecond

	rollBackSucessPeriod = 150 * time.Millisecond
)

var (
	testTimeoutProvider = timeoutProvider{
		upgraderStartGracePeriod:        100 * time.Millisecond,
		upgraderStuckInSameStateTimeout: 100 * time.Millisecond,
		stateReconcilerPollInterval:     10 * time.Millisecond,
		absoluteNoProgressTimeout:       absoluteNoProgressTimeout,
		rollbackSuccessPeriod:           rollBackSucessPeriod,
	}
)

type fakeClusterStorage struct {
	lock   sync.Mutex
	values map[string]*storage.ClusterUpgradeStatus
}

func (f *fakeClusterStorage) UpdateClusterUpgradeStatus(_ context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	if _, ok := f.values[clusterID]; !ok {
		return errors.Errorf("WRITE TO UNEXPECTED ID %s", clusterID)
	}
	f.values[clusterID] = status.Clone()
	return nil
}

func (f *fakeClusterStorage) GetCluster(_ context.Context, id string) (*storage.Cluster, bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if value, ok := f.values[id]; ok {
		return &storage.Cluster{MainImage: "stackrox.io/main", Status: &storage.ClusterStatus{UpgradeStatus: value}}, true, nil
	}
	return nil, false, nil
}

func newFakeClusterStorage(existingIDs ...string) *fakeClusterStorage {
	m := make(map[string]*storage.ClusterUpgradeStatus)
	for _, id := range existingIDs {
		m[id] = nil
	}
	return &fakeClusterStorage{values: m}
}

type recordingConn struct {
	lock     sync.Mutex
	triggers []*central.SensorUpgradeTrigger

	returnErr bool
}

func (*recordingConn) CheckAutoUpgradeSupport() error {
	return nil
}

func (r *recordingConn) InjectMessage(_ concurrency.Waitable, msg *central.MsgToSensor) error {
	if r.returnErr {
		return errors.New("RETURNING FAKE ERR FROM INJECTMESSAGE ON REQUEST")
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.triggers = append(r.triggers, msg.GetSensorUpgradeTrigger().Clone())
	return nil
}

func (r *recordingConn) InjectMessageIntoQueue(_ *central.MsgFromSensor) {}

func (r *recordingConn) getSentTriggers() []*central.SensorUpgradeTrigger {
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := make([]*central.SensorUpgradeTrigger, 0, len(r.triggers))
	copied = append(copied, r.triggers...)
	return copied
}

type UpgradeCtrlTestSuite struct {
	suite.Suite

	autoTriggerFlag *concurrency.Flag
	storage         *fakeClusterStorage
	upgradeCtrl     UpgradeController
	conn            *recordingConn
	cancelSensorCtx context.CancelFunc
}

type connWithVersion struct {
	*recordingConn
	version string
}

func (c connWithVersion) SensorVersion() string {
	return c.version
}

func TestUpgradeCtrl(t *testing.T) {
	suite.Run(t, new(UpgradeCtrlTestSuite))
}

func (suite *UpgradeCtrlTestSuite) validateTrigger(trigger *central.SensorUpgradeTrigger) {
	suite.NotEmpty(trigger.GetUpgradeProcessId())
	suite.NotEmpty(trigger.GetCommand())
	suite.Equal("stackrox.io/main:"+fakeCurrVersion, trigger.GetImage())
	suite.processMustBeActiveWithID(trigger.GetUpgradeProcessId())
}

func (suite *UpgradeCtrlTestSuite) createUpgradeCtrl() {
	suite.autoTriggerFlag = new(concurrency.Flag)
	suite.conn = new(recordingConn)
	suite.storage = newFakeClusterStorage(fakeClusterID)
	var err error
	suite.upgradeCtrl, err = newWithTimeoutProvider(fakeClusterID, suite.storage, suite.autoTriggerFlag, testTimeoutProvider)
	suite.Require().NoError(err)
}

func (suite *UpgradeCtrlTestSuite) waitForTriggerNumber(numExpected int) *central.SensorUpgradeTrigger {
	poller := concurrency.NewPoller(func() bool {
		return len(suite.conn.getSentTriggers()) >= numExpected
	}, 10*time.Millisecond)
	suite.True(concurrency.WaitWithTimeout(poller, time.Second))
	triggers := suite.conn.getSentTriggers()
	suite.Len(triggers, numExpected)
	return triggers[numExpected-1]
}

func (suite *UpgradeCtrlTestSuite) createSensorCtx() context.Context {
	incomingCtx, cancel := context.WithCancel(context.Background())
	suite.cancelSensorCtx = cancel
	return incomingCtx
}

func (suite *UpgradeCtrlTestSuite) registerConnectionFromNonAncientSensorVersion(version string) {
	errSig := suite.upgradeCtrl.RegisterConnection(suite.createSensorCtx(), connWithVersion{suite.conn, version})
	suite.NotNil(errSig)
	suite.False(concurrency.IsDone(errSig))
}

func (suite *UpgradeCtrlTestSuite) getUpgradeStatus() *storage.ClusterUpgradeStatus {
	cluster, _, err := suite.storage.GetCluster(context.Background(), fakeClusterID)
	suite.Require().NoError(err)
	upgradeStatus := cluster.GetStatus().GetUpgradeStatus()
	suite.Require().NotNil(upgradeStatus)
	return upgradeStatus
}

func (suite *UpgradeCtrlTestSuite) upgradabilityMustBe(upgradability storage.ClusterUpgradeStatus_Upgradability) {
	suite.Equal(upgradability, suite.getUpgradeStatus().GetUpgradability())
}

func (suite *UpgradeCtrlTestSuite) upgradeStateMustBe(upgradeState storage.UpgradeProgress_UpgradeState) {
	suite.True(concurrency.PollWithTimeout(func() bool {
		return upgradeState == suite.getUpgradeStatus().GetMostRecentProcess().GetProgress().GetUpgradeState()
	}, 10*time.Millisecond, time.Second), "Got state %s but expected %s", suite.getUpgradeStatus().GetMostRecentProcess().GetProgress().GetUpgradeState(), upgradeState)
	if stateutils.TerminalStates.Contains(upgradeState) {
		suite.False(suite.getUpgradeStatus().GetMostRecentProcess().GetActive())
	}
}

func (suite *UpgradeCtrlTestSuite) processMustBeActiveWithID(id string) {
	suite.True(suite.getUpgradeStatus().GetMostRecentProcess().GetActive())
	suite.Equal(id, suite.getUpgradeStatus().GetMostRecentProcess().GetId())
}

func (suite *UpgradeCtrlTestSuite) simulateInitiationOfAutoUpgrade() string {
	suite.autoTriggerFlag.Set(true)
	suite.registerConnectionFromNonAncientSensorVersion(fakeOldVersion)
	suite.upgradabilityMustBe(storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE)

	// It should have triggered an auto upgrade.
	triggerSent := suite.waitForTriggerNumber(1)
	suite.validateTrigger(triggerSent)
	return triggerSent.GetUpgradeProcessId()
}

func (suite *UpgradeCtrlTestSuite) upgraderCheckInAndRespMustBe(processID string, workflow string, stage sensorupgrader.Stage, expectedWorkflowResp string) {
	suite.upgraderCheckInWithErrAndRespMustBe(processID, workflow, stage, "", expectedWorkflowResp)
}

func (suite *UpgradeCtrlTestSuite) upgraderCheckInWithErrAndRespMustBe(processID string, workflow string, stage sensorupgrader.Stage, upgraderErr string, expectedWorkflowResp string) {
	resp, err := suite.upgradeCtrl.ProcessCheckInFromUpgrader(&central.UpgradeCheckInFromUpgraderRequest{
		UpgradeProcessId:       processID,
		ClusterId:              fakeClusterID,
		CurrentWorkflow:        workflow,
		LastExecutedStage:      stage.String(),
		LastExecutedStageError: upgraderErr,
	})
	suite.NoError(err)
	suite.Equal(expectedWorkflowResp, resp.GetWorkflowToExecute())
}

func (suite *UpgradeCtrlTestSuite) sensorSaysUpgraderIsUp(processID string) {
	suite.NoError(suite.upgradeCtrl.ProcessCheckInFromSensor(&central.UpgradeCheckInFromSensorRequest{
		ClusterId:        fakeClusterID,
		UpgradeProcessId: processID,
		State: &central.UpgradeCheckInFromSensorRequest_PodStates{
			PodStates: &central.UpgradeCheckInFromSensorRequest_UpgraderPodStates{
				States: []*central.UpgradeCheckInFromSensorRequest_UpgraderPodState{
					{PodName: "upgrader", Started: true},
				},
			},
		},
	}))
}

func (suite *UpgradeCtrlTestSuite) SetupTest() {
	suite.createUpgradeCtrl()
	testutils.SetMainVersion(suite.T(), fakeCurrVersion)
}

func (suite *UpgradeCtrlTestSuite) TestDoesntTriggerWithoutConnection() {
	suite.Error(suite.upgradeCtrl.Trigger(context.Background()))
}

func (suite *UpgradeCtrlTestSuite) TestHandlingNewConnectionFromAncientSensor() {
	errSig := suite.upgradeCtrl.RegisterConnection(context.Background(), connWithVersion{recordingConn: suite.conn})
	suite.Nil(errSig)

	suite.upgradabilityMustBe(storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED)
	// Make sure no messages are sent to the indicator. Sleep for a bit to be sure.
	time.Sleep(100 * time.Millisecond)
	suite.Empty(suite.conn.getSentTriggers())
}

func (suite *UpgradeCtrlTestSuite) TestWithUpToDateSensor() {
	suite.registerConnectionFromNonAncientSensorVersion(fakeCurrVersion)

	suite.upgradabilityMustBe(storage.ClusterUpgradeStatus_UP_TO_DATE)
	// It should send an empty trigger to the sensor of the current version,
	// which is a signal to clean up the upgrade process if it isn't cleaned up yet.
	suite.Equal(&central.SensorUpgradeTrigger{}, suite.waitForTriggerNumber(1))
}

func (suite *UpgradeCtrlTestSuite) TestWithOldSensorNoAutoUpgradeFlag() {
	suite.registerConnectionFromNonAncientSensorVersion(fakeOldVersion)

	suite.upgradabilityMustBe(storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE)

	// It should send an empty trigger, since we don't want the sensor to auto-upgrade
	suite.Equal(&central.SensorUpgradeTrigger{}, suite.waitForTriggerNumber(1))

	// Now, trigger an upgrade.
	suite.NoError(suite.upgradeCtrl.Trigger(context.Background()))
	triggerSent := suite.waitForTriggerNumber(2)
	suite.validateTrigger(triggerSent)

	// Shouldn't be able to re-trigger when an upgrade is in progress.
	suite.Error(suite.upgradeCtrl.Trigger(context.Background()))
}

func (suite *UpgradeCtrlTestSuite) TestWithOldSensorAndAutoUpgradeFlag() {
	suite.simulateInitiationOfAutoUpgrade()
}

func (suite *UpgradeCtrlTestSuite) TestUpgradeHappyPath() {
	processID := suite.simulateInitiationOfAutoUpgrade()
	suite.sensorSaysUpgraderIsUp(processID)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHING)
	suite.upgraderCheckInAndRespMustBe(processID, "", sensorupgrader.UnsetStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHED)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.PreflightStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.ExecuteStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.ExecuteStage, sensorupgrader.RollForwardWorkflow)
	suite.registerConnectionFromNonAncientSensorVersion(fakeCurrVersion)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_COMPLETE)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.ExecuteStage, sensorupgrader.CleanupWorkflow)
}

func (suite *UpgradeCtrlTestSuite) TestNewSensorChecksInBeforeUpgrader() {
	processID := suite.simulateInitiationOfAutoUpgrade()
	suite.sensorSaysUpgraderIsUp(processID)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHING)
	suite.upgraderCheckInAndRespMustBe(processID, "", sensorupgrader.UnsetStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHED)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.PreflightStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE)

	// The updated sensor has checked in, but the upgrader hasn't... yet. Don't jump the gun on marking the upgrade complete.
	suite.registerConnectionFromNonAncientSensorVersion(fakeCurrVersion)
	suite.upgradeStateMustBe(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE)

	// Now the upgrader checks in. Aaand we're done!
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.ExecuteStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_COMPLETE)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.ExecuteStage, sensorupgrader.CleanupWorkflow)
}

func (suite *UpgradeCtrlTestSuite) TestUpgradePreFlightFails() {
	processID := suite.simulateInitiationOfAutoUpgrade()
	suite.sensorSaysUpgraderIsUp(processID)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHING)
	suite.upgraderCheckInAndRespMustBe(processID, "", sensorupgrader.UnsetStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHED)
	suite.upgraderCheckInWithErrAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.PreflightStage, "NOO", sensorupgrader.CleanupWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_FAILED)
}

func (suite *UpgradeCtrlTestSuite) TestUpgradeExecutionErrorAndRollback() {
	processID := suite.simulateInitiationOfAutoUpgrade()
	suite.sensorSaysUpgraderIsUp(processID)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHING)
	suite.upgraderCheckInAndRespMustBe(processID, "", sensorupgrader.UnsetStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHED)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.PreflightStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE)

	// Now, simulate a termination of the current sensor connection, and an error.
	suite.cancelSensorCtx()
	// Give the context cancel a little bit of time to propagate.
	time.Sleep(100 * time.Millisecond)
	suite.upgraderCheckInWithErrAndRespMustBe(processID, sensorupgrader.RollForwardWorkflow, sensorupgrader.ExecuteStage, "NOO", sensorupgrader.RollBackWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK)
	suite.upgraderCheckInAndRespMustBe(processID, sensorupgrader.RollBackWorkflow, sensorupgrader.ExecuteStage, sensorupgrader.CleanupWorkflow)

	// Now, an old sensor checks in.
	suite.registerConnectionFromNonAncientSensorVersion(fakeOldVersion)
	time.Sleep(rollBackSucessPeriod / 3)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK)
	time.Sleep(rollBackSucessPeriod)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK)
}

// Failure cases where upgrades never start.
func (suite *UpgradeCtrlTestSuite) TestUpgradeNeverStarts() {
	suite.simulateInitiationOfAutoUpgrade()
	time.Sleep(absoluteNoProgressTimeout)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_TIMED_OUT)
}

func (suite *UpgradeCtrlTestSuite) TestUpgraderJustDisappears() {
	processID := suite.simulateInitiationOfAutoUpgrade()
	suite.sensorSaysUpgraderIsUp(processID)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHING)
	suite.upgraderCheckInAndRespMustBe(processID, "", sensorupgrader.UnsetStage, sensorupgrader.RollForwardWorkflow)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADER_LAUNCHED)
	time.Sleep(absoluteNoProgressTimeout)
	suite.upgradeStateMustBe(storage.UpgradeProgress_UPGRADE_TIMED_OUT)
}

func (suite *UpgradeCtrlTestSuite) TestSensorGoesAwayAndComesBackInTheMiddle() {
	suite.registerConnectionFromNonAncientSensorVersion(fakeOldVersion)

	suite.upgradabilityMustBe(storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE)

	// It should send an empty trigger, since we don't want the sensor to auto-upgrade
	suite.Equal(&central.SensorUpgradeTrigger{}, suite.waitForTriggerNumber(1))

	// Now, trigger an upgrade.
	suite.NoError(suite.upgradeCtrl.Trigger(context.Background()))
	triggerSent := suite.waitForTriggerNumber(2)
	suite.validateTrigger(triggerSent)

	// The sensor connection goes away...
	suite.cancelSensorCtx()
	time.Sleep(100 * time.Millisecond)

	// ... and it comes back.
	suite.registerConnectionFromNonAncientSensorVersion(fakeOldVersion)
	// It should get another trigger
	triggerSent = suite.waitForTriggerNumber(3)
	suite.validateTrigger(triggerSent)

}

// TESTS THAT DON'T USE THE SUITE ARE BELOW.

func TestUpgradeControllerDoesntInitializeIfClusterIDInvalid(t *testing.T) {
	var autoTriggerFlag concurrency.Flag
	_, err := New("DEFINITELY_NOT_FAKECLUSTERID", newFakeClusterStorage(fakeClusterID), &autoTriggerFlag)
	assert.Error(t, err)
}
