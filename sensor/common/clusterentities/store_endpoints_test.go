package clusterentities

import (
	"slices"

	"maps"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
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

			for tickNo := 0; tickNo < slices.Max(maps.Keys(tCase.lookupResultsAfterTick))+1; tickNo++ {
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
