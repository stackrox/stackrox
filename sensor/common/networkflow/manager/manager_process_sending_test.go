package manager

import (
	"strconv"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
)

func (b *sendNetflowsSuite) TestUpdateComputer_ProcessListening() {
	now := timestamp.Now()
	open := timestamp.InfiniteFuture
	closed := now.Add(-time.Second)

	const deploymentID = srcID
	p1 := defaultProcessKey()
	p2 := anotherProcessKey()

	e1p1open := createEndpointPairWithProcess(now, now, open, p1)
	e1p1closed := createEndpointPairWithProcess(now, now, closed, p1)
	e1p2open := createEndpointPairWithProcess(now, now, open, p2)

	type event struct {
		description                 string
		input                       *endpointPair
		expectedNumContainerLookups int
		expectedNumUpdatesEndpoint  int
		expectedNumUpdatesProcess   int
		expectedDeduperState        map[indicator.BinaryHash]indicator.BinaryHash
		expectedUpdatedProcesses    *indicator.ProcessInfo
	}

	tt := map[string]struct {
		events      []event
		plopEnabled bool
	}{
		"open-e1p1 followed by close-e1p1 should yield empty deduper": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  1,
					expectedNumUpdatesProcess:   1,
					expectedDeduperState: map[indicator.BinaryHash]indicator.BinaryHash{
						e1p1open.endpointIndicator(deploymentID).BinaryKey(): e1p1open.processListeningIndicator().BinaryKey(),
					},
					expectedUpdatedProcesses: &p1,
				},
				{
					description:                 "Closing endpoint e1 with new process p1",
					input:                       e1p1closed,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  1,
					expectedNumUpdatesProcess:   1,
					expectedDeduperState:        map[indicator.BinaryHash]indicator.BinaryHash{},
					expectedUpdatedProcesses:    &p1,
				},
			},
			plopEnabled: true,
		},
		"open-e1p1 followed by close-e1p1 on disabled PLoP should yield no updates and empty deduper": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  1,
					expectedNumUpdatesProcess:   0,
					expectedDeduperState: map[string]string{
						e1p1open.endpointIndicator(deploymentID).Key(indicator.HashingAlgoHash): "",
					},
					expectedUpdatedProcesses: nil,
				},
				{
					description:                 "Closing endpoint e1 with new process p1",
					input:                       e1p1closed,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  1,
					expectedNumUpdatesProcess:   0,
					expectedDeduperState:        map[string]string{},
					expectedUpdatedProcesses:    nil,
				},
			},
			plopEnabled: false,
		},
		"open-e1p1 followed by open-e1p2 should not keep p1 in deduper (replacing behavior)": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  1,
					expectedNumUpdatesProcess:   1,
					expectedDeduperState: map[indicator.BinaryHash]indicator.BinaryHash{
						e1p1open.endpointIndicator(deploymentID).BinaryKey(): e1p1open.processListeningIndicator().BinaryKey(),
					},
					expectedUpdatedProcesses: &p1,
				},
				{
					description:                 "Open the same endpoint e1 with new process p2",
					input:                       e1p2open,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  0,
					expectedNumUpdatesProcess:   1,
					expectedDeduperState: map[indicator.BinaryHash]indicator.BinaryHash{
						e1p2open.endpointIndicator(deploymentID).BinaryKey(): e1p2open.processListeningIndicator().BinaryKey(),
					},
					expectedUpdatedProcesses: &p2,
				},
			},
			plopEnabled: true,
		},
		"duplicated inputs should not yield duplicated updates": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  1,
					expectedNumUpdatesProcess:   1,
					expectedUpdatedProcesses:    &p1,
				},
				{
					description:                 "Open the same endpoint e1 with the same process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdatesEndpoint:  0,
					expectedNumUpdatesProcess:   0,
					expectedDeduperState: map[indicator.BinaryHash]indicator.BinaryHash{
						e1p1open.endpointIndicator(deploymentID).BinaryKey(): e1p1open.processListeningIndicator().BinaryKey(),
					},
					expectedUpdatedProcesses: nil,
				},
			},
			plopEnabled: true,
		},
	}

	for name, tc := range tt {
		b.Run(name, func() {
			b.uc.ResetState()
			b.T().Setenv(env.ProcessesListeningOnPort.EnvVar(), strconv.FormatBool(tc.plopEnabled))

			for i, e := range tc.events {
				b.T().Logf("event[%d]: %s", i, e.description)
				b.expectContainerLookups(e.expectedNumContainerLookups)
				b.updateEp(e.input)
				b.thenTickerTicks()
				// Calculate total number of expected updates to Central in this tick
				expectedNumMessagesToCentral := e.expectedNumUpdatesEndpoint + e.expectedNumUpdatesProcess
				updatesP, updatesE := b.getUpdates(expectedNumMessagesToCentral)
				b.T().Logf("event[%d]: got updatesP: %v", i, updatesP)
				b.T().Logf("event[%d]: got updatesE: %v", i, updatesE)
				b.printDedupers()
				b.assertNoOtherUpdates()

				b.Require().Equal(e.expectedNumUpdatesProcess, len(updatesP), "Number of process updates should match")
				b.Require().Equal(e.expectedNumUpdatesEndpoint, len(updatesE), "Number of endpoint updates should match")

				if e.expectedDeduperState != nil {
					b.assertDeduperState(e.expectedDeduperState)
				}

				if e.expectedNumUpdatesProcess > 0 && e.expectedUpdatedProcesses != nil {
					b.Equal(e.expectedUpdatedProcesses.ProcessName, updatesP[0].GetProcess().GetProcessName(), "Updated process name should match")
					b.Equal(e.expectedUpdatedProcesses.ProcessArgs, updatesP[0].GetProcess().GetProcessArgs(), "Updated process args should match")
					b.Equal(e.expectedUpdatedProcesses.ProcessExec, updatesP[0].GetProcess().GetProcessExecFilePath(), "Updated process exec should match")
				} else {
					b.Equal(0, len(updatesP), "Number of process updates should be 0 when 0 is expected")
				}
			}
		})
	}
}

func (b *sendNetflowsSuite) assertNoOtherUpdates() {
	// No other updates
	mustNotRead(b.T(), b.m.sensorUpdates)
}

func (b *sendNetflowsSuite) assertDeduperState(expected map[indicator.BinaryHash]indicator.BinaryHash) {
	if testable, ok := b.uc.(updatecomputer.TestableUpdateComputer); ok {
		testable.WithEndpointDeduperAccess(func(epDeduper map[indicator.BinaryHash]indicator.BinaryHash) {
			b.EqualValuesf(expected, epDeduper, "deduper state should match expected")
		})
	} else {
		b.T().Skip("Update computer doesn't support deduper state access")
	}
}

func (b *sendNetflowsSuite) printDedupers() {
	if testable, ok := b.uc.(updatecomputer.TestableUpdateComputer); ok {
		testable.WithEndpointDeduperAccess(func(deduper map[indicator.BinaryHash]indicator.BinaryHash) {
			b.T().Logf("endpoint->process deduper: (%v)", deduper)
		})
	} else {
		b.T().Log("Update computer doesn't support deduper state access")
	}
}

func (b *sendNetflowsSuite) getUpdates(num int) ([]*storage.ProcessListeningOnPortFromSensor, []*storage.NetworkEndpoint) {
	p := make([]*storage.ProcessListeningOnPortFromSensor, 0)
	e := make([]*storage.NetworkEndpoint, 0)
	for range num {
		msg := mustReadTimeout(b.T(), b.m.sensorUpdates)
		switch msg.Msg.(type) {
		case *central.MsgFromSensor_ProcessListeningOnPortUpdate:
			update := msg.GetMsg().(*central.MsgFromSensor_ProcessListeningOnPortUpdate)
			p = append(p, update.ProcessListeningOnPortUpdate.GetProcessesListeningOnPorts()...)
		case *central.MsgFromSensor_NetworkFlowUpdate:
			update := msg.GetMsg().(*central.MsgFromSensor_NetworkFlowUpdate)
			e = append(e, update.NetworkFlowUpdate.GetUpdatedEndpoints()...)
		default:
			b.T().Logf("unexpected msg: %v", msg.Msg)
		}

	}
	return p, e
}
