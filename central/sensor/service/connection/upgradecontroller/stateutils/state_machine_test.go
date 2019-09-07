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
	return
}

func TestStateMachineHandlesAllUpgraderStartupCases(t *testing.T) {
	for _, upgradeState := range getAllUpgradeStates() {
		assert.NotNil(t, computeNextStateAndResp(upgradeState, "", sensorupgrader.UnsetStage, ""), "Value was nil for %s", upgradeState)
	}
}

func TestStateMachineForRollForwardAndRollBackCases(t *testing.T) {
	for _, wf := range []string{sensorupgrader.RollForwardWorkflow, sensorupgrader.RollBackWorkflow} {
		stages := sensorupgrader.Workflows()[wf]
		for _, stage := range stages {
			for _, upgradeState := range getAllUpgradeStates() {
				for _, upgraderErr := range []string{"", "FAKE ERR"} {
					assert.NotNil(t, computeNextStateAndResp(upgradeState, wf, stage, upgraderErr), "Value was nil for %s/%s/%s/%s", upgradeState, wf, stage, upgraderErr)
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

func (s *stageStorage) get() storage.UpgradeProgress_UpgradeState {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.stateHistory[len(s.stateHistory)-1]
}

func (s *stageStorage) set(state storage.UpgradeProgress_UpgradeState) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.stateHistory = append(s.stateHistory, state)
}

func setupStoreAndMockUpgrader(t *testing.T, errPairs []workflowStagePair) (*stageStorage, *mockUpgrader) {
	store := &stageStorage{
		stateHistory: []storage.UpgradeProgress_UpgradeState{storage.UpgradeProgress_UPGRADER_LAUNCHING},
	}

	m := &mockUpgrader{
		nextWorkflow: func(workflow string, stage sensorupgrader.Stage, upgraderErr string) string {
			nextState, nextWorkflow, _ := DetermineNextStateAndWorkflowForUpgrader(store.get(), workflow, stage, upgraderErr)
			store.set(nextState)
			return nextWorkflow
		},
		errPairs: errPairs,
	}
	go m.run()

	enoughRunsPoller := concurrency.NewPoller(func() bool {
		return m.done.Get() || m.getNumRunsDone() > 10
	}, 50*time.Millisecond)
	assert.True(t, concurrency.WaitWithTimeout(enoughRunsPoller, time.Second), m.getNumRunsDone())

	return store, m
}

func TestStateMachineWithMockUpgraderHappyPath(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, nil)
	store.set(storage.UpgradeProgress_UPGRADE_COMPLETE)
	donePoller := concurrency.NewPoller(m.done.Get, 10*time.Millisecond)
	assert.True(t, concurrency.WaitWithTimeout(donePoller, time.Second))
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_OPERATIONS_DONE,
		storage.UpgradeProgress_UPGRADE_COMPLETE,
	}, sliceutils.Unique(store.stateHistory))
}

func TestStateMachineWithMockUpgraderFailBeforePreflightChecks(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
		{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.FetchBundleStage},
	})
	assert.True(t, m.done.Get())
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_UPGRADE_INITIALIZATION_ERROR,
	}, sliceutils.Unique(store.stateHistory))
}

func TestStateMachineWithMockUpgraderPreFlightChecksFail(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
		{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.PreflightStage},
	})
	assert.True(t, m.done.Get())
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_FAILED,
	}, sliceutils.Unique(store.stateHistory))
}

func TestStateMachineWithMockUpgraderExecutionFailed(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
		{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.ExecuteStage},
	})
	assert.True(t, m.done.Get())
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK,
	}, sliceutils.Unique(store.stateHistory))
}

func TestStateMachineWithMockUpgraderExecutionFailedAndRollbackFailed(t *testing.T) {
	store, m := setupStoreAndMockUpgrader(t, []workflowStagePair{
		{workflow: sensorupgrader.RollForwardWorkflow, stage: sensorupgrader.ExecuteStage},
		{workflow: sensorupgrader.RollBackWorkflow, stage: sensorupgrader.ExecuteStage},
	})
	assert.True(t, m.done.Get())
	assert.Equal(t, []storage.UpgradeProgress_UpgradeState{
		storage.UpgradeProgress_UPGRADER_LAUNCHING,
		storage.UpgradeProgress_UPGRADER_LAUNCHED,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_COMPLETE,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLING_BACK,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLBACK_FAILED,
	}, sliceutils.Unique(store.stateHistory))
}
