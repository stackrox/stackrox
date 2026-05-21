package clusterentities

import (
	"maps"
	"slices"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
)

type expectation struct {
	query            net.NumericEndpoint
	wantLocation     whereThingIsStored
	wantLookupResult LookupResult
}

func (s *ClusterEntitiesStoreTestSuite) TestMemoryAboutPastEndpoints() {
	cases := map[string]struct {
		numTicksToRemember uint16
		entityUpdates      map[int][]eUpdate // tick -> updates
		// operationAfterTick defines tick IDs after which an operation should be simulated
		// (e.g., deletion of a container, or going offline).
		operationAfterTick     map[int]operation
		lookupResultsAfterTick map[int][]expectation
		publicIPsAtCleanup     []string
	}{
		"Memory disabled with no reset should remember endpoint forever": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{}, // do not reset at all
			lookupResultsAfterTick: map[int][]expectation{
				// before tick 1: endpoint should be added immediately
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: endpoint should stay there forever
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				2: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
			},
		},
		"Deployment changing IP should populate history with the previous IP": {
			numTicksToRemember: 1,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
				2: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.2",
						port:         80,
						portName:     "http",
						incremental:  false, // overwrite
					},
				},
			},
			operationAfterTick: map[int]operation{}, // do not reset at all
			lookupResultsAfterTick: map[int][]expectation{
				// before tick 1: endpoint should be added immediately
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: no change expected
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 2: endpoint should be overwritten and old IP placed in history
				2: {
					expectDeployment80("10.0.0.1", "depl1", "http", history),
					expectDeployment80("10.0.0.2", "depl1", "http", theMap),
				},
				// after tick 3: history expires
				3: {
					expectDeployment80("10.0.0.1", "depl1", "http", nowhere),
					expectDeployment80("10.0.0.2", "depl1", "http", theMap),
				},
			},
		},
		"Re-add deployment previously marked as historical while still in history": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
				3: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: deleteDeployment1},
			lookupResultsAfterTick: map[int][]expectation{
				// before tick 1: endpoint should be added immediately
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: no change expected
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// deletion happens here
				// after tick 2: endpoint should be marked as historical
				2: {expectDeployment80("10.0.0.1", "depl1", "http", history)},
				// after tick 3: endpoint should be no longer considered historical
				3: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
			},
		},
		"Re-add deployment previously marked as historical before history expired": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
				4: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: deleteDeployment1},
			lookupResultsAfterTick: map[int][]expectation{
				// before tick 1: endpoint should be added immediately
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: no change expected
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// deletion happens here
				// after tick 2: endpoint should be marked as historical
				2: {expectDeployment80("10.0.0.1", "depl1", "http", history)},
				// after tick 3: endpoint should still be in the history
				3: {expectDeployment80("10.0.0.1", "depl1", "http", history)},
				// after tick 4: endpoint should be no longer considered historical
				4: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
			},
		},
		"Re-add deployment previously marked as historical after history expired": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
				5: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: deleteDeployment1},
			lookupResultsAfterTick: map[int][]expectation{
				// before tick 1: endpoint should be added immediately
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: no change expected
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// deletion happens here
				// after tick 2: endpoint should be marked as historical
				2: {expectDeployment80("10.0.0.1", "depl1", "http", history)},
				// after tick 3: endpoint should still be in the history
				3: {expectDeployment80("10.0.0.1", "depl1", "http", history)},
				// after tick 4: endpoint should be no longer considered historical
				4: {expectDeployment80("10.0.0.1", "depl1", "http", nowhere)},
				// after tick 5: endpoint should be added again
				5: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
			},
		},
		"IP changing the owner deployment should populate history with previous IP": {
			numTicksToRemember: 1,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
				2: {
					{
						deploymentID: "depl2",
						containerID:  "pod2",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  false, // overwrite
					},
				},
			},
			operationAfterTick: map[int]operation{},
			lookupResultsAfterTick: map[int][]expectation{
				// before tick 1: endpoint should be added immediately
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: no change expected
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				// after tick 1: endpoint should be overwritten and old IP placed in history
				2: {
					expectDeployment80("10.0.0.1", "depl1", "http", history),
					expectDeployment80("10.0.0.1", "depl2", "http", theMap),
				},
				// after tick 2: history expires
				3: {
					expectDeployment80("10.0.0.1", "depl1", "http", nowhere),
					expectDeployment80("10.0.0.1", "depl2", "http", theMap),
				},
			},
		},
		"IP changing owner with memory disabled should not retain previous deployment in history": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  true,
					},
				},
				2: {
					{
						deploymentID: "depl2",
						containerID:  "pod2",
						ipAddr:       "10.0.0.1",
						port:         80,
						portName:     "http",
						incremental:  false, // overwrite
					},
				},
			},
			operationAfterTick: map[int]operation{},
			lookupResultsAfterTick: map[int][]expectation{
				0: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				1: {expectDeployment80("10.0.0.1", "depl1", "http", theMap)},
				2: {
					expectDeployment80("10.0.0.1", "depl1", "http", nowhere),
					expectDeployment80("10.0.0.1", "depl2", "http", theMap),
				},
				3: {
					expectDeployment80("10.0.0.1", "depl1", "http", nowhere),
					expectDeployment80("10.0.0.1", "depl2", "http", theMap),
				},
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			store := NewStore(tCase.numTicksToRemember, nil, true)
			ipListener := newTestPublicIPsListener(s.T())
			store.RegisterPublicIPsListener(ipListener)
			defer func() {
				s.True(store.UnregisterPublicIPsListener(ipListener))
				s.Equalf(len(tCase.publicIPsAtCleanup), ipListener.data.Cardinality(),
					"the listeners of public IPs have incorrect data at test cleanup")
				for gotIP := range ipListener.data {
					s.Contains(tCase.publicIPsAtCleanup, gotIP.AsNetIP().String())
				}
			}()

			ticks := slices.Max(slices.Collect(maps.Keys(tCase.lookupResultsAfterTick))) + 1
			for tickNo := 0; tickNo < ticks; tickNo++ {
				expectations := tCase.lookupResultsAfterTick[tickNo]
				// Add entities to the store (mimic data arriving from the K8s informers)
				if updatesForTick, ok := tCase.entityUpdates[tickNo]; ok {
					for _, update := range updatesForTick {
						store.Apply(map[string]*EntityData{
							update.deploymentID: entityUpdateWithPortName(update.ipAddr, update.containerID, update.port, update.portName),
						}, update.incremental)
					}
				}
				// Assert on endpoints
				s.T().Logf("Endpoints (tick %d):\n%s", tickNo, store.endpointsStore.String())
				if len(expectations) == 0 {
					s.T().Fatalf("No expectations for tick %d - test case may miss assertions", tickNo)
				}
				for _, want := range expectations {
					current, historical, ipCurrent, ipHistorical := store.endpointsStore.lookupEndpoint(want.query, store.podIPsStore)
					switch want.wantLocation {
					case theMap:
						s.Containsf(append(current, ipCurrent...), want.wantLookupResult, "expected to find endpoint %q in the map in tick %d", want.query.IPAndPort.String(), tickNo)
						s.NotContainsf(append(historical, ipHistorical...), want.wantLookupResult, "expected NOT to find endpoint %q in history in tick %d", want.query.IPAndPort.String(), tickNo)
					case history:
						s.NotContainsf(append(current, ipCurrent...), want.wantLookupResult, "expected NOT to find endpoint %q in the map in tick %d", want.query.IPAndPort.String(), tickNo)
						s.Containsf(append(historical, ipHistorical...), want.wantLookupResult, "expected to find endpoint %q in history in tick %d", want.query.IPAndPort.String(), tickNo)
					case nowhere:
						s.NotContainsf(append(current, ipCurrent...), want.wantLookupResult, "expected NOT to find endpoint %q in the map in tick %d", want.query.IPAndPort.String(), tickNo)
						s.NotContainsf(append(historical, ipHistorical...), want.wantLookupResult, "expected NOT to find endpoint %q in history in tick %d", want.query.IPAndPort.String(), tickNo)
					}
				}
				store.RecordTick()
				if op, ok := tCase.operationAfterTick[tickNo]; ok {
					s.T().Logf("Exec operation=%s (tick %d). State after operation:", op, tickNo)
					switch op {
					case mapReset:
						store.resetMaps()
					case deleteDeployment1:
						// This is how a DELETE operation is implemented
						store.Apply(map[string]*EntityData{"depl1": {}}, false)
					}
					s.T().Logf("\t\tEndpoints (tick %d): %s", tickNo, store.endpointsStore.String())

				}
			}
		})
	}
}

func expectDeployment80(ip, deplID, portName string, location whereThingIsStored) expectation {
	// some values are hardcoded for simplicity
	return buildExpectation(ip, deplID, portName, location, networkgraph.EntityForDeployment, 80)
}

func buildExpectation(ip, deplID, portName string, location whereThingIsStored, typeFunc func(string) networkgraph.Entity, port int) expectation {
	return expectation{
		query:        buildEndpoint(ip, uint16(port)),
		wantLocation: location,
		wantLookupResult: LookupResult{
			Entity:         typeFunc(deplID),
			ContainerPorts: []uint16{uint16(port)},
			PortNames:      []string{portName},
		},
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestEmptyEndpointUpdatePreservesSeenState() {
	ep := buildEndpoint("10.0.0.1", 80)
	httpTarget := EndpointTargetInfo{ContainerPort: 80, PortName: "http"}

	type applyStep struct {
		deploymentID string
		data         *EntityData
		incremental  bool
	}
	applyWithoutEndpoints := func(deplID string) applyStep {
		d := &EntityData{}
		d.AddContainerID("ctr-"+deplID, ContainerMetadata{DeploymentID: deplID})
		return applyStep{deploymentID: deplID, data: d, incremental: false}
	}
	endpointUpdate := func(deplID string) applyStep {
		d := &EntityData{}
		d.AddEndpoint(ep, httpTarget)
		return applyStep{deploymentID: deplID, data: d, incremental: true}
	}
	endpointOverwrite := func(deplID string) applyStep {
		d := &EntityData{}
		d.AddEndpoint(ep, httpTarget)
		return applyStep{deploymentID: deplID, data: d, incremental: false}
	}

	deplAResult := LookupResult{
		Entity:         networkgraph.EntityForDeployment("deplA"),
		ContainerPorts: []uint16{80},
		PortNames:      []string{"http"},
	}
	deplBResult := LookupResult{
		Entity:         networkgraph.EntityForDeployment("deplB"),
		ContainerPorts: []uint16{80},
		PortNames:      []string{"http"},
	}

	cases := map[string]struct {
		steps     []applyStep
		wantDeplA whereThingIsStored
		wantDeplB whereThingIsStored
	}{
		"deployment seen with empty endpoints should not trigger takeover when it later acquires real endpoints": {
			steps: []applyStep{
				applyWithoutEndpoints("deplA"),
				endpointUpdate("deplB"),
				endpointOverwrite("deplA"),
			},
			wantDeplA: theMap,
			wantDeplB: theMap,
		},
		"repeated empty applies should not corrupt the seen marker": {
			steps: []applyStep{
				applyWithoutEndpoints("deplA"),
				applyWithoutEndpoints("deplA"),
				applyWithoutEndpoints("deplA"),
				endpointUpdate("deplB"),
				endpointOverwrite("deplA"),
			},
			wantDeplA: theMap,
			wantDeplB: theMap,
		},
		"unchanged real endpoints after empty start should hit the fast path without side effects": {
			steps: []applyStep{
				applyWithoutEndpoints("deplA"),
				endpointUpdate("deplB"),
				endpointOverwrite("deplA"),
				endpointOverwrite("deplA"),
			},
			wantDeplA: theMap,
			wantDeplB: theMap,
		},
		"two deployments both seen empty should not trigger takeover when they share the same endpoint": {
			steps: []applyStep{
				applyWithoutEndpoints("deplA"),
				applyWithoutEndpoints("deplB"),
				endpointOverwrite("deplA"),
				endpointOverwrite("deplB"),
			},
			wantDeplA: theMap,
			wantDeplB: theMap,
		},
		"known deployment that goes empty through diff replace should keep seen marker": {
			steps: []applyStep{
				endpointOverwrite("deplA"),
				applyWithoutEndpoints("deplA"),
				endpointUpdate("deplB"),
				endpointOverwrite("deplA"),
			},
			wantDeplA: theMap,
			wantDeplB: theMap,
		},
		"unseen deployment should trigger takeover as expected": {
			steps: []applyStep{
				endpointUpdate("deplB"),
				endpointOverwrite("deplA"),
			},
			wantDeplA: theMap,
			wantDeplB: history,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			store := NewStore(5, nil, true)
			for _, step := range tc.steps {
				store.Apply(map[string]*EntityData{step.deploymentID: step.data}, step.incremental)
			}
			current, historical, _, _ := store.endpointsStore.lookupEndpoint(ep, store.podIPsStore)
			for _, want := range []struct {
				result   LookupResult
				location whereThingIsStored
				label    string
			}{
				{deplAResult, tc.wantDeplA, "deplA"},
				{deplBResult, tc.wantDeplB, "deplB"},
			} {
				switch want.location {
				case theMap:
					s.Contains(current, want.result, "%s should be current", want.label)
					s.NotContains(historical, want.result, "%s should not be historical", want.label)
				case history:
					s.NotContains(current, want.result, "%s should not be current", want.label)
					s.Contains(historical, want.result, "%s should be historical", want.label)
				case nowhere:
					s.NotContains(current, want.result, "%s should not be current", want.label)
					s.NotContains(historical, want.result, "%s should not be historical", want.label)
				}
			}
		})
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestEndpointTakeoverFastPathDoesNotPolluteReverseHistory() {
	store := NewStore(5, nil, true)

	ep1 := buildEndpoint("10.0.0.1", 80)
	ep2 := buildEndpoint("10.0.0.2", 80)

	deplA := &EntityData{}
	deplA.AddEndpoint(ep1, EndpointTargetInfo{ContainerPort: 80, PortName: "http"})
	deplA.AddEndpoint(ep2, EndpointTargetInfo{ContainerPort: 80, PortName: "http"})
	store.Apply(map[string]*EntityData{"deplA": deplA}, true)

	deplB := &EntityData{}
	deplB.AddEndpoint(ep1, EndpointTargetInfo{ContainerPort: 80, PortName: "http"})
	store.Apply(map[string]*EntityData{"deplB": deplB}, true)

	reverseHistDeplA, ok := store.endpointsStore.reverseHistoricalEndpoints["deplA"]
	s.True(ok, "deplA should have history entry after endpoint takeover")
	s.Contains(reverseHistDeplA, ep1, "taken-over endpoint must be historical for previous owner")
	s.NotContains(reverseHistDeplA, ep2, "unrelated endpoint must not be marked historical during single-endpoint takeover")
}

func buildEntityData(entries map[net.NumericEndpoint][]EndpointTargetInfo) *EntityData {
	data := &EntityData{}
	for endpoint, targetInfos := range entries {
		for _, targetInfo := range targetInfos {
			data.AddEndpoint(endpoint, targetInfo)
		}
	}
	return data
}

func (s *ClusterEntitiesStoreTestSuite) TestEndpointsUnchangedNoLockVariantsMatchExpectedSemantics() {
	ep1 := buildEndpoint("10.0.0.1", 80)
	ep2 := buildEndpoint("10.0.0.2", 80)

	cases := map[string]struct {
		current       *EntityData
		next          map[net.NumericEndpoint][]EndpointTargetInfo
		wantUnchanged bool
	}{
		"empty current and empty next are unchanged": {
			next:          map[net.NumericEndpoint][]EndpointTargetInfo{},
			wantUnchanged: true,
		},
		"empty current and non-empty next are changed": {
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
			},
			wantUnchanged: false,
		},
		"unchanged endpoints and target infos": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
					{ContainerPort: 443, PortName: "https"},
				},
				ep2: {
					{ContainerPort: 8080, PortName: "metrics"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 443, PortName: "https"},
					{ContainerPort: 80, PortName: "http"},
				},
				ep2: {
					{ContainerPort: 8080, PortName: "metrics"},
				},
			},
			wantUnchanged: true,
		},
		"same target info on distinct endpoints is unchanged": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
				ep2: {
					{ContainerPort: 80, PortName: "http"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
				ep2: {
					{ContainerPort: 80, PortName: "http"},
				},
			},
			wantUnchanged: true,
		},
		"duplicate target infos are treated as changed": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
					{ContainerPort: 80, PortName: "http"},
				},
			},
			wantUnchanged: false,
		},
		"duplicate target infos can mask removal and are treated as changed": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
					{ContainerPort: 443, PortName: "https"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
					{ContainerPort: 80, PortName: "http"},
				},
			},
			wantUnchanged: false,
		},
		"same length but different target info is treated as changed": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
					{ContainerPort: 443, PortName: "https"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
					{ContainerPort: 9090, PortName: "metrics"},
				},
			},
			wantUnchanged: false,
		},
		"missing endpoint is treated as changed": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
				ep2: {
					{ContainerPort: 8080, PortName: "metrics"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
			},
			wantUnchanged: false,
		},
		"extra endpoint is treated as changed": {
			current: buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
			}),
			next: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {
					{ContainerPort: 80, PortName: "http"},
				},
				ep2: {
					{ContainerPort: 8080, PortName: "metrics"},
				},
			},
			wantUnchanged: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			store := newEndpointsStoreWithMemory(5)
			if c.current != nil {
				store.applyNoLock(map[string]*EntityData{"depl": c.current}, false)
			}

			s.Equal(c.wantUnchanged, store.endpointsUnchangedNoLock("depl", c.next))
		})
	}
}

// TestDiffReplaceNoLock exercises the per-endpoint diff path in replaceNoLock.
// Each case starts from an initial state, applies a non-incremental update, and
// verifies that endpoints end up in the correct location (current map, history, or nowhere).
func (s *ClusterEntitiesStoreTestSuite) TestDiffReplaceNoLock() {
	ep1 := buildEndpoint("10.0.0.1", 80)
	ep2 := buildEndpoint("10.0.0.2", 80)
	ep3 := buildEndpoint("10.0.0.3", 80)
	httpTarget := EndpointTargetInfo{ContainerPort: 80, PortName: "http"}
	httpsTarget := EndpointTargetInfo{ContainerPort: 443, PortName: "https"}
	metricsTarget := EndpointTargetInfo{ContainerPort: 9090, PortName: "metrics"}

	mkExpect := func(ip string, containerPort uint16, portName string, loc whereThingIsStored) expectation {
		return expectation{
			query:        buildEndpoint(ip, 80),
			wantLocation: loc,
			wantLookupResult: LookupResult{
				Entity:         networkgraph.EntityForDeployment("depl"),
				ContainerPorts: []uint16{containerPort},
				PortNames:      []string{portName},
			},
		}
	}

	cases := map[string]struct {
		memorySize   uint16
		initial      map[net.NumericEndpoint][]EndpointTargetInfo
		update       map[net.NumericEndpoint][]EndpointTargetInfo
		expectations []expectation
		// afterTicks maps tick count → expectations checked after that many
		// RecordTick calls. Used to verify history expiry behavior.
		afterTicks map[int][]expectation
	}{
		"partial change: one endpoint modified, others unchanged": {
			memorySize: 5,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {httpTarget},
				ep3: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpsTarget},
				ep2: {httpTarget},
				ep3: {httpTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 443, "https", theMap),
				mkExpect("10.0.0.2", 80, "http", theMap),
				mkExpect("10.0.0.3", 80, "http", theMap),
			},
		},
		"endpoint removed: should move to history": {
			memorySize: 5,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 80, "http", theMap),
				mkExpect("10.0.0.2", 80, "http", history),
			},
		},
		"endpoint added: should appear in current map": {
			memorySize: 5,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {metricsTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 80, "http", theMap),
				mkExpect("10.0.0.2", 9090, "metrics", theMap),
			},
		},
		"target info changed on same endpoint: old target info moves to history": {
			memorySize: 5,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpsTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 443, "https", theMap),
				mkExpect("10.0.0.1", 80, "http", history),
			},
		},
		"target info changed: old info expires from history after configured ticks": {
			memorySize: 2,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpsTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 443, "https", theMap),
				mkExpect("10.0.0.1", 80, "http", history),
			},
			afterTicks: map[int][]expectation{
				1: {
					mkExpect("10.0.0.1", 443, "https", theMap),
					mkExpect("10.0.0.1", 80, "http", history),
				},
				2: {
					mkExpect("10.0.0.1", 443, "https", theMap),
					mkExpect("10.0.0.1", 80, "http", nowhere),
				},
			},
		},
		"endpoint removed: history expires after configured ticks": {
			memorySize: 2,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 80, "http", theMap),
				mkExpect("10.0.0.2", 80, "http", history),
			},
			afterTicks: map[int][]expectation{
				1: {
					mkExpect("10.0.0.1", 80, "http", theMap),
					mkExpect("10.0.0.2", 80, "http", history),
				},
				2: {
					mkExpect("10.0.0.1", 80, "http", theMap),
					mkExpect("10.0.0.2", 80, "http", nowhere),
				},
			},
		},
		"all endpoints removed: should all move to history": {
			memorySize: 5,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {httpTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{},
			expectations: []expectation{
				mkExpect("10.0.0.1", 80, "http", history),
				mkExpect("10.0.0.2", 80, "http", history),
			},
		},
		"all endpoints unchanged: no history pollution": {
			memorySize: 5,
			initial: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {httpsTarget},
			},
			update: map[net.NumericEndpoint][]EndpointTargetInfo{
				ep1: {httpTarget},
				ep2: {httpsTarget},
			},
			expectations: []expectation{
				mkExpect("10.0.0.1", 80, "http", theMap),
				mkExpect("10.0.0.2", 443, "https", theMap),
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			store := NewStore(tc.memorySize, nil, true)
			store.Apply(map[string]*EntityData{"depl": buildEntityData(tc.initial)}, false)
			store.Apply(map[string]*EntityData{"depl": buildEntityData(tc.update)}, false)

			for _, want := range tc.expectations {
				current, historical, _, _ := store.endpointsStore.lookupEndpoint(want.query, store.podIPsStore)
				switch want.wantLocation {
				case theMap:
					s.Containsf(current, want.wantLookupResult, "expected endpoint %s in current", want.query.IPAndPort.String())
					s.NotContainsf(historical, want.wantLookupResult, "expected endpoint %s NOT in history", want.query.IPAndPort.String())
				case history:
					s.NotContainsf(current, want.wantLookupResult, "expected endpoint %s NOT in current", want.query.IPAndPort.String())
					s.Containsf(historical, want.wantLookupResult, "expected endpoint %s in history", want.query.IPAndPort.String())
				case nowhere:
					s.NotContainsf(current, want.wantLookupResult, "expected endpoint %s NOT in current", want.query.IPAndPort.String())
					s.NotContainsf(historical, want.wantLookupResult, "expected endpoint %s NOT in history", want.query.IPAndPort.String())
				}
			}

			lastTick := slices.Max(append(slices.Collect(maps.Keys(tc.afterTicks)), 0))
			for t := range lastTick {
				tick := t + 1
				store.RecordTick()
				for _, want := range tc.afterTicks[tick] {
					current, historical, _, _ := store.endpointsStore.lookupEndpoint(want.query, store.podIPsStore)
					switch want.wantLocation {
					case theMap:
						s.Containsf(current, want.wantLookupResult, "after tick %d: expected endpoint %s in current", tick, want.query.IPAndPort.String())
						s.NotContainsf(historical, want.wantLookupResult, "after tick %d: expected endpoint %s NOT in history", tick, want.query.IPAndPort.String())
					case history:
						s.NotContainsf(current, want.wantLookupResult, "after tick %d: expected endpoint %s NOT in current", tick, want.query.IPAndPort.String())
						s.Containsf(historical, want.wantLookupResult, "after tick %d: expected endpoint %s in history", tick, want.query.IPAndPort.String())
					case nowhere:
						s.NotContainsf(current, want.wantLookupResult, "after tick %d: expected endpoint %s NOT in current", tick, want.query.IPAndPort.String())
						s.NotContainsf(historical, want.wantLookupResult, "after tick %d: expected endpoint %s NOT in history", tick, want.query.IPAndPort.String())
					}
				}
			}
		})
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestApplyCanonicalizesDuplicateTargetInfosWhenTheyMaskRemoval() {
	ep := buildEndpoint("10.0.0.3", 8080)
	httpTarget := EndpointTargetInfo{ContainerPort: 8080, PortName: "http"}
	httpsTarget := EndpointTargetInfo{ContainerPort: 8443, PortName: "https"}

	store := newEndpointsStoreWithMemory(5)
	store.applyNoLock(map[string]*EntityData{
		"depl": buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
			ep: {httpTarget, httpsTarget},
		}),
	}, false)

	store.applyNoLock(map[string]*EntityData{
		"depl": buildEntityData(map[net.NumericEndpoint][]EndpointTargetInfo{
			ep: {httpTarget, httpTarget},
		}),
	}, false)

	targetInfos := store.endpointMap[ep]["depl"]
	s.Len(targetInfos, 1)
	s.True(targetInfos.Contains(httpTarget))
	s.False(targetInfos.Contains(httpsTarget))
}

func (s *ClusterEntitiesStoreTestSuite) TestTargetInfoUnchangedNoLock() {
	ep := buildEndpoint("10.0.0.1", 8080)
	deplID := "depl-1"

	ti1 := EndpointTargetInfo{ContainerPort: 8080, PortName: "http"}
	ti2 := EndpointTargetInfo{ContainerPort: 8443, PortName: "https"}
	ti3 := EndpointTargetInfo{ContainerPort: 9090, PortName: "metrics"}

	cases := map[string]struct {
		stored []EndpointTargetInfo
		input  []EndpointTargetInfo
		want   bool
	}{
		"should return true when both are empty": {
			stored: nil,
			input:  nil,
			want:   true,
		},
		"should return true when nothing stored and input is empty slice": {
			stored: nil,
			input:  []EndpointTargetInfo{},
			want:   true,
		},
		"should return false when nothing stored but input is non-empty": {
			stored: nil,
			input:  []EndpointTargetInfo{ti1},
			want:   false,
		},
		"should return false when stored but input is empty": {
			stored: []EndpointTargetInfo{ti1},
			input:  []EndpointTargetInfo{},
			want:   false,
		},
		"should return true for single identical element": {
			stored: []EndpointTargetInfo{ti1},
			input:  []EndpointTargetInfo{ti1},
			want:   true,
		},
		"should return false for single different element": {
			stored: []EndpointTargetInfo{ti1},
			input:  []EndpointTargetInfo{ti2},
			want:   false,
		},
		"should return true for multiple elements same order": {
			stored: []EndpointTargetInfo{ti1, ti2, ti3},
			input:  []EndpointTargetInfo{ti1, ti2, ti3},
			want:   true,
		},
		"should return true for multiple elements different order": {
			stored: []EndpointTargetInfo{ti1, ti2, ti3},
			input:  []EndpointTargetInfo{ti3, ti1, ti2},
			want:   true,
		},
		"should return false when lengths differ": {
			stored: []EndpointTargetInfo{ti1, ti2},
			input:  []EndpointTargetInfo{ti1, ti2, ti3},
			want:   false,
		},
		"should return false when one element differs": {
			stored: []EndpointTargetInfo{ti1, ti2},
			input:  []EndpointTargetInfo{ti1, ti3},
			want:   false,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			store := newEndpointsStoreWithMemory(5)
			if len(tc.stored) > 0 {
				store.endpointMap[ep] = map[string]set.Set[EndpointTargetInfo]{
					deplID: set.NewSet(tc.stored...),
				}
			}
			got := store.targetInfoUnchangedNoLock(deplID, ep, tc.input)
			s.Equal(tc.want, got)
		})
	}
}
