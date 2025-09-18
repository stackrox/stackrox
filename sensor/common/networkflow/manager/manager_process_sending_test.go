package manager

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
)

func (b *sendNetflowsSuite) TestUpdateComputer_ProcessListening() {
	now := timestamp.Now()
	open := timestamp.InfiniteFuture
	closed := now.Add(-time.Second)

	p1 := defaultProcessKey()
	p2 := anotherProcessKey()

	e1p1open := createEndpointPairWithProcess(now, now, open, p1)
	e1p1closed := createEndpointPairWithProcess(now, now, closed, p1)
	e1p2open := createEndpointPairWithProcess(now, now, open, p2)

	type event struct {
		description                 string
		input                       *endpointPair
		expectedNumContainerLookups int
		expectedNumUpdates          map[updatecomputer.EnrichedEntity]int
		expectedDeduperLen          map[updatecomputer.EnrichedEntity]int
		deduperShouldContain        map[updatecomputer.EnrichedEntity]string
		deduperShouldNotContain     map[updatecomputer.EnrichedEntity]string
		expectedUpdatedProcesses    *indicator.ProcessInfo
	}

	tt := map[string]struct {
		events []event
	}{
		"open-e1p1 followed by close-e1p1 should yield empty deduper": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdates: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					expectedDeduperLen: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					deduperShouldContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p1open.deduperKey(indicator.HashingAlgoHash),
					},
					deduperShouldNotContain:  map[updatecomputer.EnrichedEntity]string{},
					expectedUpdatedProcesses: &p1,
				},
				{
					description:                 "Closing endpoint e1 with new process p1",
					input:                       e1p1closed,
					expectedNumContainerLookups: 1,
					expectedNumUpdates: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					expectedDeduperLen: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  0,
						updatecomputer.EndpointEnrichedEntity: 0,
					},
					deduperShouldContain: map[updatecomputer.EnrichedEntity]string{},
					deduperShouldNotContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p1open.deduperKey(indicator.HashingAlgoHash),
					},
					expectedUpdatedProcesses: &p1,
				},
			},
		},
		"open-e1p1 followed by open-e1p2 should not keep p1 in deduper (replacing behavior)": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdates: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					expectedDeduperLen: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					deduperShouldContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p1open.deduperKey(indicator.HashingAlgoHash),
					},
					deduperShouldNotContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p2open.deduperKey(indicator.HashingAlgoHash),
					},
					expectedUpdatedProcesses: &p1,
				},
				{
					description:                 "Open the same endpoint e1 with new process p2",
					input:                       e1p2open,
					expectedNumContainerLookups: 1,
					expectedNumUpdates: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 0,
					},
					expectedDeduperLen: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					deduperShouldContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p2open.deduperKey(indicator.HashingAlgoHash),
					},
					deduperShouldNotContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p1open.deduperKey(indicator.HashingAlgoHash),
					},
					expectedUpdatedProcesses: &p2,
				},
			},
		},
		"duplicated inputs should not yield duplicated updates": {
			events: []event{
				{
					description:                 "Open endpoint e1 with new process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdates: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					expectedDeduperLen: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					deduperShouldContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p1open.deduperKey(indicator.HashingAlgoHash),
					},
					deduperShouldNotContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p2open.deduperKey(indicator.HashingAlgoHash),
					},
					expectedUpdatedProcesses: &p1,
				},
				{
					description:                 "Open the same endpoint e1 with the same process p1",
					input:                       e1p1open,
					expectedNumContainerLookups: 1,
					expectedNumUpdates: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  0,
						updatecomputer.EndpointEnrichedEntity: 0,
					},
					expectedDeduperLen: map[updatecomputer.EnrichedEntity]int{
						updatecomputer.ProcessEnrichedEntity:  1,
						updatecomputer.EndpointEnrichedEntity: 1,
					},
					deduperShouldContain: map[updatecomputer.EnrichedEntity]string{
						updatecomputer.ProcessEnrichedEntity: e1p1open.deduperKey(indicator.HashingAlgoHash),
					},
					deduperShouldNotContain:  map[updatecomputer.EnrichedEntity]string{},
					expectedUpdatedProcesses: nil,
				},
			},
		},
	}

	for name, tc := range tt {
		b.Run(name, func() {
			b.uc.ResetState()
			for i, e := range tc.events {
				b.T().Logf("event[%d]: %s", i, e.description)
				b.expectContainerLookups(e.expectedNumContainerLookups)
				b.updateEp(e.input)
				b.thenTickerTicks()
				// Calculate total number of expected updates to Central in this tick
				expectedNumMessagesToCentral := 0
				for _, value := range e.expectedNumUpdates {
					expectedNumMessagesToCentral += value
				}
				updatesP, updatesE := b.getUpdates(expectedNumMessagesToCentral)
				b.T().Logf("event[%d]: got updatesP: %v", i, updatesP)
				b.T().Logf("event[%d]: got updatesE: %v", i, updatesE)
				b.printDedupers()
				b.assertNoOtherUpdates()

				b.Require().Equal(e.expectedNumUpdates[updatecomputer.ProcessEnrichedEntity], len(updatesP), "Number of process updates should match")
				b.Require().Equal(e.expectedNumUpdates[updatecomputer.EndpointEnrichedEntity], len(updatesE), "Number of endpoint updates should match")

				for ee, l := range e.expectedDeduperLen {
					b.assertDeduperLen(ee, l)
				}

				for ee, key := range e.deduperShouldContain {
					b.assertDeduperContains(ee, key)
				}

				for ee, key := range e.deduperShouldNotContain {
					b.assertDeduperNotContains(ee, key)
				}
				if e.expectedNumUpdates[updatecomputer.ProcessEnrichedEntity] > 0 && e.expectedUpdatedProcesses != nil {
					b.Equal(e.expectedUpdatedProcesses.ProcessName, updatesP[0].GetProcess().GetProcessName(), "Updated process name should match")
					b.Equal(e.expectedUpdatedProcesses.ProcessArgs, updatesP[0].GetProcess().GetProcessArgs(), "Updated process args should match")
					b.Equal(e.expectedUpdatedProcesses.ProcessExec, updatesP[0].GetProcess().GetProcessExecFilePath(), "Updated process exec should match")
				}
			}
		})
	}
}

func (b *sendNetflowsSuite) assertNoOtherUpdates() {
	// No other updates
	mustNotRead(b.T(), b.m.sensorUpdates)
}

func (b *sendNetflowsSuite) assertDeduperLen(ee updatecomputer.EnrichedEntity, expectedLen int) {
	dedupers := b.uc.GetState()
	got := dedupers[ee].Cardinality()
	b.Equal(expectedLen, got, "expected %d deduper entries, got=%d", expectedLen, got)
}

func (b *sendNetflowsSuite) assertDeduperContains(ee updatecomputer.EnrichedEntity, value string) {
	dedupers := b.uc.GetState()
	b.Truef(dedupers[ee].Contains(value), "expected %s deduper to contain %s", ee, value)
}

func (b *sendNetflowsSuite) assertDeduperNotContains(ee updatecomputer.EnrichedEntity, value string) {
	dedupers := b.uc.GetState()
	b.Falsef(dedupers[ee].Contains(value), "expected %s deduper to not contain %s", ee, value)
}

func (b *sendNetflowsSuite) printDedupers() {
	dedupers := b.uc.GetState()
	b.T().Logf("conn deduper: (%s)", dedupers[updatecomputer.ConnectionEnrichedEntity].ElementsString(";"))
	b.T().Logf("endp deduper: (%s)", dedupers[updatecomputer.EndpointEnrichedEntity].ElementsString(";"))
	b.T().Logf("proc deduper: (%s)", dedupers[updatecomputer.ProcessEnrichedEntity].ElementsString(";"))
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
