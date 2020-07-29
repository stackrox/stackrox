package stateutils

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func getAllUpgradeStates() (states []storage.UpgradeProgress_UpgradeState) {
	for upgradeStateIdx := range storage.UpgradeProgress_UpgradeState_name {
		upgradeState := storage.UpgradeProgress_UpgradeState(upgradeStateIdx)
		states = append(states, upgradeState)
	}
	return states
}

func getAllUpgradeTypes() (upgradeTypes []storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType) {
	for upgradeTypeIdx := range storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType_name {
		upgradeTypes = append(upgradeTypes, storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType(upgradeTypeIdx))
	}
	return upgradeTypes
}

func TestStateMachineHandlesAllUpgraderStartupCases(t *testing.T) {
	for _, upgradeType := range getAllUpgradeTypes() {
		for _, upgradeState := range getAllUpgradeStates() {
			assert.NotNil(t, computeNextStateAndResp(upgradeType, upgradeState, "", sensorupgrader.UnsetStage, ""), "Value was nil for %s", upgradeState)
		}
	}
}

func TestStateMachineForRollForwardAndRollBackCases(t *testing.T) {
	for _, upgradeType := range getAllUpgradeTypes() {
		for _, wf := range []string{sensorupgrader.RollForwardWorkflow, sensorupgrader.RollBackWorkflow} {
			stages := sensorupgrader.Workflows()[wf]
			for _, stage := range stages {
				for _, upgradeState := range getAllUpgradeStates() {
					for _, upgraderErr := range []string{"", "FAKE ERR"} {
						assert.NotNil(t, computeNextStateAndResp(upgradeType, upgradeState, wf, stage, upgraderErr), "Value was nil for %s/%s/%s/%s", upgradeState, wf, stage, upgraderErr)
					}
				}
			}
		}
	}
}

type workflowStagePair struct {
	workflow string
	stage    sensorupgrader.Stage
}

type mockUpgrader struct {
	errPairs     []workflowStagePair
	nextWorkflow func(workflow string, stage sensorupgrader.Stage, upgraderErr string) string

	done concurrency.Flag

	numRunsDone int32
}

func (m *mockUpgrader) incrNumRunsDone() {
	atomic.AddInt32(&m.numRunsDone, 1)
}

func (m *mockUpgrader) getNumRunsDone() int {
	return int(atomic.LoadInt32(&m.numRunsDone))
}

// A simplified version of the logic we expect the upgrader to perform.
func (m *mockUpgrader) run() {
	workflow := ""
	stage := sensorupgrader.UnsetStage
	err := ""

	for {
		nextWorkflow := m.nextWorkflow(workflow, stage, err)
		var nextStage sensorupgrader.Stage
		var nextErr string
		if nextWorkflow != workflow {
			// New workflow
			nextStage = sensorupgrader.Workflows()[nextWorkflow][0]
		} else {
			idx := sliceutils.Find(sensorupgrader.Workflows()[workflow], stage)
			if idx == -1 {
				panic(fmt.Sprintf("UNEXPECTED: workflow: %s, stage: %s", workflow, stage))
			}
			if idx < len(sensorupgrader.Workflows()[workflow])-1 {
				idx++
			}
			nextStage = sensorupgrader.Workflows()[workflow][idx]
		}

		for _, errPair := range m.errPairs {
			if errPair.workflow == nextWorkflow && errPair.stage == nextStage {
				nextErr = "FAKEERR"
			}
		}

		if workflow == sensorupgrader.CleanupWorkflow && stage == sensorupgrader.WaitForDeletionStage {
			m.done.Set(true)
			return
		}

		workflow = nextWorkflow
		stage = nextStage
		err = nextErr
		time.Sleep(10 * time.Millisecond)
		m.incrNumRunsDone()
	}
}

type stageStorage struct {
	stateHistory []storage.UpgradeProgress_UpgradeState
	lock         sync.Mutex
}

func (s *stageStorage) getFullHistory() []storage.UpgradeProgress_UpgradeState {
	s.lock.Lock()
	defer s.lock.Unlock()
	return append([]storage.UpgradeProgress_UpgradeState{}, s.stateHistory...)
}

func (s *stageStorage) set(state storage.UpgradeProgress_UpgradeState) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.setNoLock(state)
}

func (s *stageStorage) setNoLock(state storage.UpgradeProgress_UpgradeState) {
	s.stateHistory = append(s.stateHistory, state)
}

func setupStoreAndMockUpgrader(t *testing.T, errPairs []workflowStagePair, upgradeType storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType) (*stageStorage, *mockUpgrader) {
	store := &stageStorage{
		stateHistory: []storage.UpgradeProgress_UpgradeState{storage.UpgradeProgress_UPGRADER_LAUNCHING},
	}

	m := &mockUpgrader{
		nextWorkflow: func(workflow string, stage sensorupgrader.Stage, upgraderErr string) string {
			store.lock.Lock()
			defer store.lock.Unlock()
			nextState, nextWorkflow, _ := DetermineNextStateAndWorkflowForUpgrader(upgradeType, store.stateHistory[len(store.stateHistory)-1], workflow, stage, upgraderErr)
			store.setNoLock(nextState)
			return nextWorkflow
		},
		errPairs: errPairs,
	}
	go m.run()

	assert.True(t, concurrency.PollWithTimeout(func() bool {
		return m.done.Get() || m.getNumRunsDone() > 10
	}, 50*time.Millisecond, time.Second), m.getNumRunsDone())

	return store, m
}

func assertMockUpgraderDone(t *testing.T, m *mockUpgrader) {
	assert.True(t, concurrency.PollWithTimeout(m.done.Get, 50*time.Millisecond, time.Second))
}

func TestStateMachineWithMockUpgraderHappyPath(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, nil, storage.ClusterUpgradeStatus_UpgradeProcessStatus_UPGRADE)
	store.set(storage.UpgradeProgress_UPGRADE_COMPLETE)
	assert.True(t, concurrency.PollWithTimeout(m.done.Get, 10*time.Millisecond, time.Second))
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE,
		storage.UpgradeProgress_UPGRADE_COMPLETE,
	}, sliceutils.Unique(store.getFullHistory()))
}

func TestStateMachineWithMockUpgraderCertRotationHappyPath(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, nil, storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION)
	assert.True(t, concurrency.PollWithTimeout(m.done.Get, 10*time.Millisecond, time.Second))
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_COMPLETE,
	}, sliceutils.Unique(store.getFullHistory()))
}

func TestStateMachineWithMockUpgraderFailBeforePreflightChecks(t *testing.T) {
	for _, upgradeType := range getAllUpgradeTypes() {
		t.Run(upgradeType.String(), func(t *testing.T) {
			store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
				{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.FetchBundleStage},
			}, upgradeType)
			assertMockUpgraderDone(t, m)
			assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
				storage.UpgradeProgress_UPGRADER_LAUNCHING,
				storage.UpgradeProgress_UPGRADER_LAUNCHED,
				storage.UpgradeProgress_UPGRADE_INITIALIZATION_ERROR,
			}, sliceutils.Unique(store.getFullHistory()))
		})
	}
}

func TestStateMachineWithMockUpgraderPreFlightChecksFail(t *testing.T) {
	for _, upgradeType := range getAllUpgradeTypes() {
		t.Run(upgradeType.String(), func(t *testing.T) {
			store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
				{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.PreflightStage},
			}, upgradeType)
			assertMockUpgraderDone(t, m)
			assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
				storage.UpgradeProgress_UPGRADER_LAUNCHING,
				storage.UpgradeProgress_UPGRADER_LAUNCHED,
				storage.UpgradeProgress_PRE_FLIGHT_CHECKS_FAILED,
			}, sliceutils.Unique(store.getFullHistory()))
		})
	}
}

func TestStateMachineWithMockUpgraderExecutionFailed(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
		{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.ExecuteStage},
	}, storage.ClusterUpgradeStatus_UpgradeProcessStatus_UPGRADE)
	assertMockUpgraderDone(t, m)
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK,
	}, sliceutils.Unique(store.getFullHistory()))
}

func TestStateMachineWithMockUpgraderExecutionFailedCertRotation(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
		{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.ExecuteStage},
	}, storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION)
	assertMockUpgraderDone(t, m)
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK,
	}, sliceutils.Unique(store.getFullHistory()))
}

func TestStateMachineWithMockUpgraderExecutionFailedAndRollbackFailed(t *testing.T) {
	for _, upgradeType := range getAllUpgradeTypes() {
		t.Run(upgradeType.String(), func(t *testing.T) {

			store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
				{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.ExecuteStage},
				{workflow: sensorupgrader.RollBackWorkflow, stage: sensorupgrader.ExecuteStage},
			}, upgradeType)
			assertMockUpgraderDone(t, m)
			assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
				storage.UpgradeProgress_UPGRADER_LAUNCHING,
				storage.UpgradeProgress_UPGRADER_LAUNCHED,
				storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
				storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK,
				storage.UpgradeProgress_UPGRADE_ERROR_ROLLBACK_FAILED,
			}, sliceutils.Unique(store.getFullHistory()))
		})
	}
}
