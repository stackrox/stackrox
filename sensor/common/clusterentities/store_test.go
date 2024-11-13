package clusterentities

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/maps"
)

func TestClusterEntitiesStore(t *testing.T) {
	suite.Run(t, new(ClusterEntitiesStoreTestSuite))
}

type ClusterEntitiesStoreTestSuite struct {
	suite.Suite
}

// eUpdate represents a request to the entity store to append, or replace an entry
type eUpdate struct {
	deploymentID string
	containerID  string
	incremental  bool
}

// region external-entities test

func buildEndpoint(ip string) net.NumericEndpoint {
	return net.NumericEndpoint{
		IPAndPort: net.NetworkPeerID{
			Address: net.ParseIP(ip),
		},
		L4Proto: net.TCP,
	}
}

func entityUpdate(ip, contID string, port uint16) *EntityData {
	ed := &EntityData{}
	ep := buildEndpoint(ip)
	ed.AddEndpoint(ep, EndpointTargetInfo{
		ContainerPort: port,
		PortName:      "ehlo",
	})
	ed.AddIP(ep.IPAndPort.Address)
	ed.AddContainerID(contID, ContainerMetadata{
		DeploymentID:  "",
		DeploymentTS:  0,
		PodID:         "",
		PodUID:        "",
		ContainerName: "name-of-" + contID,
		ContainerID:   contID,
		Namespace:     "",
		StartTime:     nil,
		ImageID:       "",
	})
	return ed
}

func (s *ClusterEntitiesStoreTestSuite) TestMemoryAboutPast() {
	type eUpdate struct {
		deploymentID string
		containerID  string
		ipAddr       string
		port         uint16
		incremental  bool
	}
	cases := map[string]struct {
		numTicksToRemember    uint16
		entityUpdates         map[int][]eUpdate // tick -> updates
		endpointsAfterTick    []map[string]bool
		containerIDsAfterTick []map[string]bool
	}{
		"Memory disabled should forget 10.0.0.1 immediately": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						incremental:  true, // append
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.3.0.1",
						port:         80,
						incremental:  false, // replace
					},
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": false, "10.3.0.1": true}, // pre-tick 1: 10.0.0.1 should be overwritten immediately - only 10.3.0.1 should exist
				{"10.0.0.1": false, "10.3.0.1": true}, // tick 1: only 10.3.0.1 should exist
				{"10.0.0.1": false, "10.3.0.1": true}, // tick 2: only 10.3.0.1 should exist
			},
			containerIDsAfterTick: []map[string]bool{
				{"pod1": true}, // before tick 1: container should be added immediately
				{"pod1": true}, // tick 1: update of IP should not cause the container ID to disappear
				{"pod1": true}, // tick 2: nothing has happened that would cause the container ID to disappear
			},
		},
		"Old IPs should be gone on the first tick": {
			numTicksToRemember: 1,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.3.0.1",
						port:         80,
						incremental:  false,
					},
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true, "10.3.0.1": true},  // pre-tick 1: both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 1: only 10.3.0.1 should exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 2: only 10.3.0.1 should exist
			},
			containerIDsAfterTick: []map[string]bool{
				{"pod1": true}, // before tick 1
				{"pod1": true}, // tick 1
				{"pod1": true}, // tick 2
			},
		},
		"Updates of the same IP should not expire": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						incremental:  false,
					},
				},
				2: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						incremental:  false,
					},
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true}, // tick 0: update0
				{"10.0.0.1": true}, // tick 1: mark update0 as historical
				{"10.0.0.1": true}, // tick 2: historical update0 exists; add again in update2
				{"10.0.0.1": true}, // tick 3: historical update0 would be deleted, but update2 shall exist
				{"10.0.0.1": true}, // tick 4: update2 must exist
				{"10.0.0.1": true}, // tick 5: update2 must exist
			},
			containerIDsAfterTick: []map[string]bool{
				{"pod1": true},
				{"pod1": true},
				{"pod1": true},
				{"pod1": true},
				{"pod1": true},
				{"pod1": true},
			},
		},
		"Old IPs should be gone on the 2nd tick": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.3.0.1",
						port:         80,
						incremental:  false,
					},
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true, "10.3.0.1": true},  // pre-tick 1: both must exist
				{"10.0.0.1": true, "10.3.0.1": true},  // after-tick 1: both must exist
				{"10.0.0.1": false, "10.3.0.1": true}, // after-tick 2: only 10.3.0.1 should exist
			},
			containerIDsAfterTick: []map[string]bool{
				{"pod1": true},
				{"pod1": true},
				{"pod1": true},
			},
		},
		"Old IPs should be gone for selected pods only": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.0.0.1",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "10.3.0.1",
						port:         80,
						incremental:  false,
					},
					{
						deploymentID: "depl2",
						containerID:  "pod2",
						ipAddr:       "20.0.0.1",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl2",
						containerID:  "pod2",
						ipAddr:       "20.3.0.1",
						port:         80,
						incremental:  true,
					},
				},
			},
			endpointsAfterTick: []map[string]bool{
				{"10.0.0.1": true, "10.3.0.1": true, "20.0.0.1": true, "20.3.0.1": true},
				{"10.0.0.1": true, "10.3.0.1": true, "20.0.0.1": true, "20.3.0.1": true},
				{"10.0.0.1": false, "10.3.0.1": true, "20.0.0.1": true, "20.3.0.1": true},
			},
			containerIDsAfterTick: []map[string]bool{
				{"pod1": true, "pod2": true},
				{"pod1": true, "pod2": true},
				{"pod1": true, "pod2": true},
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			entityStore := NewStoreWithMemory(tCase.numTicksToRemember)

			require.Equalf(s.T(), len(tCase.containerIDsAfterTick), len(tCase.endpointsAfterTick),
				"this test requires expected endpoints and expected container IDs to be specified for all ticks")

			for tickNo, expectation := range tCase.endpointsAfterTick {
				// Add entities to the store (mimic data arriving from the K8s informers)
				if updatesForTick, ok := tCase.entityUpdates[tickNo]; ok {
					for _, update := range updatesForTick {
						entityStore.Apply(map[string]*EntityData{
							update.deploymentID: entityUpdate(update.ipAddr, update.containerID, update.port),
						}, update.incremental)
					}
				}
				// Assert on IPs
				s.T().Logf("Historical IPs (tick %d): %v", tickNo, prettyPrintHistoricalData(entityStore.historicalIPs))
				s.T().Logf("All IPs (tick %d): %v", tickNo, maps.Keys(entityStore.ipMap))
				for endpoint, shallExist := range expectation {
					result := entityStore.LookupByEndpoint(buildEndpoint(endpoint))
					if shallExist {
						s.True(len(result) > 0, "Should find endpoint %q in tick %d. Result: %v", endpoint, tickNo, result)
					} else {
						s.True(len(result) == 0, "Should not find endpoint %q in tick %d.  Result: %v", endpoint, tickNo, result)
					}
				}

				// Assert on container IDs
				s.T().Logf("Historical container IDs (tick %d): %s", tickNo, prettyPrintHistoricalData(entityStore.historicalContainerIDs))
				s.T().Logf("All container IDs (tick %d): %v", tickNo, maps.Keys(entityStore.containerIDMap))
				for contID, shallExists := range tCase.containerIDsAfterTick[tickNo] {
					result, found := entityStore.LookupByContainerID(contID)
					if shallExists {
						s.Truef(found, "expected to find contID %q in tick %d", contID, tickNo)
						s.Equalf(contID, result.ContainerID, "Expected the result to have contID %q in tick %d. Result: %v", contID, tickNo, result)
					} else {
						s.Require().Falsef(found, "expected not to find contID %q in tick %d", contID, tickNo)
						s.Empty(result.ContainerID)
					}
				}
				entityStore.RecordTick()
			}

		})
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestChangingIPsAndExternalEntities() {
	entityStore := NewStore()
	type eUpdate struct {
		deploymentID string
		ipAddr       string
		port         uint16
		incremental  bool
	}
	cases := map[string]struct {
		entityUpdates     []eUpdate
		expectedEndpoints []string
	}{
		"Incremental updates to the store shall not loose data": {
			entityUpdates: []eUpdate{
				{
					deploymentID: "pod1",
					ipAddr:       "10.0.0.1",
					port:         80,
					incremental:  true,
				},
				{
					deploymentID: "pod1",
					ipAddr:       "10.3.0.1",
					port:         80,
					incremental:  true,
				},
			},
			expectedEndpoints: []string{"10.0.0.1", "10.3.0.1"},
		},
		"Non-incremental updates to the store shall overwrite all data for a key": {
			entityUpdates: []eUpdate{
				{
					deploymentID: "pod1",
					ipAddr:       "10.0.0.1",
					port:         80,
					incremental:  true,
				},
				{
					deploymentID: "pod1",
					ipAddr:       "10.3.0.1",
					port:         80,
					incremental:  false,
				},
				{
					deploymentID: "pod2",
					ipAddr:       "10.0.0.2",
					port:         80,
					incremental:  true,
				},
			},
			expectedEndpoints: []string{"10.3.0.1", "10.0.0.2"},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			for _, update := range tCase.entityUpdates {
				entityStore.Apply(map[string]*EntityData{
					update.deploymentID: entityUpdate(update.ipAddr, update.deploymentID, update.port),
				}, update.incremental)
			}
			for _, expectedEndpoint := range tCase.expectedEndpoints {
				result := entityStore.LookupByEndpoint(buildEndpoint(expectedEndpoint))
				s.Lenf(result, 1, "Expected endpoint %q not found", expectedEndpoint)
			}
		})
	}
}

// endregion
// region container-id history test

func (s *ClusterEntitiesStoreTestSuite) TestMemoryAboutPastContainerIDs() {
	type whereContainerIDisStored string
	const (
		// the container will be found in history
		history whereContainerIDisStored = "history"
		// the container will be found in the containerIDMap
		theMap whereContainerIDisStored = "the-map"
		// the container will not be found
		nowhere whereContainerIDisStored = "nowhere"
	)

	type operation string
	const (
		mapReset operation = "mapReset"
		// for simplicity of the test, we assume that all delete request will be for depl1
		deleteDeployment1 operation = "deleteDeployment1"
	)

	cases := map[string]struct {
		numTicksToRemember uint16
		entityUpdates      map[int][]eUpdate // tick -> updates
		// operationAfterTick defines tick IDs after which an operation should be simulated
		// (e.g., deletion of a container, or going offline).
		operationAfterTick    map[int]operation
		containerIDsAfterTick []map[string]whereContainerIDisStored
	}{
		"Memory disabled with no reset should remember pod1 forever": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{}, // do not reset at all
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1: no reset - should be in the map forever
				{"pod1": theMap},
			},
		},
		"Memory disabled with no reset and container overwrite should remember pod1 forever": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true, // append
					},
				},
				3: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  false, // delete and add
					},
				},
			},
			operationAfterTick: map[int]operation{}, // do not reset at all
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1: no reset - should be in the map forever
				{"pod1": theMap}, // after tick 2
				// container is overwritten in the map
				{"pod1": theMap}, // after tick 3: should be still in the map
				{"pod1": theMap}, // after tick 3: should be still in the map
			},
		},
		"Memory disabled with reset after tick 1 should make pod1 be forgotten before tick 2": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: mapReset},
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1: no reset yet, so it should be in the map
				// reset
				{"pod1": nowhere}, // after tick 2: should be gone
			},
		},
		"Memory for 2 ticks with reset after tick 1 should make pod1 be forgotten before tick 4": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: mapReset},
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// reset
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				{"pod1": nowhere}, // after tick 4: history expired - should be forgotten
			},
		},
		"Re-adding successfully forgotten container should reset the history status": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
				5: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: mapReset},
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// reset
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				{"pod1": nowhere}, // after tick 4: history expired - should be forgotten
				{"pod1": theMap},  // after tick 5: re-added pod1 should be findable from now on until the next reset
				{"pod1": theMap},  // after tick 6: no further reset was planned, so we should find pod1 forever
				{"pod1": theMap},
				{"pod1": theMap},
				{"pod1": theMap},
			},
		},
		"Re-adding (with overwrite) successfully forgotten container should reset the history status": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
				5: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  false, // overwrite
					},
				},
			},
			operationAfterTick: map[int]operation{1: mapReset},
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// reset
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				{"pod1": nowhere}, // after tick 4: history expired - should be forgotten
				{"pod1": theMap},  // after tick 5: re-added pod1 should be findable from now on until the next reset
				{"pod1": theMap},  // after tick 6: no further reset was planned, so we should find pod1 forever
				{"pod1": theMap},
				{"pod1": theMap},
				{"pod1": theMap},
			},
		},
		"Container is deleted normally": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: deleteDeployment1},
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// container deletion
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				{"pod1": nowhere}, // after tick 4: history expired - should be forgotten forever
				{"pod1": nowhere},
			},
		},
		"Re-adding normally deleted container should reset the history status": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
				5: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: deleteDeployment1},
			containerIDsAfterTick: []map[string]whereContainerIDisStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// container deletion
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				{"pod1": nowhere}, // after tick 4: history expired - should be forgotten forever
				// adding container again
				{"pod1": theMap}, // after tick 5: should be normally added to the map
				{"pod1": theMap}, // after tick 6: should stay in the map until the next deletion or reset
				{"pod1": theMap},
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			entityStore := NewStoreWithMemory(tCase.numTicksToRemember)

			for tickNo, expectation := range tCase.containerIDsAfterTick {
				// Add entities to the store (mimic data arriving from the K8s informers)
				if updatesForTick, ok := tCase.entityUpdates[tickNo]; ok {
					for _, update := range updatesForTick {
						entityStore.Apply(map[string]*EntityData{
							update.deploymentID: entityUpdate("10.0.0.1", update.containerID, 80),
						}, update.incremental)
					}
				}
				// Assert on container IDs
				s.T().Logf("Historical container IDs (tick %d): %s", tickNo, prettyPrintHistoricalData(entityStore.historicalContainerIDs))
				s.T().Logf("All container IDs (tick %d): %v", tickNo, maps.Keys(entityStore.containerIDMap))
				for contID, whereFound := range expectation {
					result, found := entityStore.LookupByContainerID(contID)
					resultMap, foundMap := entityStore.lookupByContainerIDNoLock(contID)
					resultHist, foundHist := entityStore.lookupByContainerIDInHistoryNoLock(contID)
					switch whereFound {
					case theMap:
						s.Truef(found, "expected to find contID %q in general in tick %d", contID, tickNo)
						s.Equalf(contID, result.ContainerID, "Expected the general result to have contID %q in tick %d. Result: %v", contID, tickNo, result)

						s.Truef(foundMap, "expected to find contID %q in the map in tick %d", contID, tickNo)
						s.Equalf(contID, resultMap.ContainerID, "Expected the map result to have contID %q in tick %d. Result: %v", contID, tickNo, resultMap)

						s.Falsef(foundHist, "expected not to find contID %q in the history in tick %d", contID, tickNo)
						s.Empty(resultHist.ContainerID)
					case history:
						s.Truef(found, "expected to find contID %q in general in tick %d", contID, tickNo)
						s.Equalf(contID, result.ContainerID, "Expected the general result to have contID %q in tick %d. Result: %v", contID, tickNo, result)

						s.Truef(foundHist, "expected to find contID %q in the history in tick %d", contID, tickNo)
						s.Equalf(contID, resultHist.ContainerID, "Expected the historical result to have contID %q in tick %d. Result: %v", contID, tickNo, resultHist)

						s.Falsef(foundMap, "expected not to find contID %q in the map in tick %d", contID, tickNo)
						s.Empty(resultMap.ContainerID)
					case nowhere:
						s.Falsef(found, "expected not to find contID %q at all in tick %d", contID, tickNo)
						s.Empty(result.ContainerID)

						s.Falsef(foundMap, "expected not to find contID %q in the map in tick %d", contID, tickNo)
						s.Empty(resultMap.ContainerID)

						s.Falsef(foundHist, "expected not to find contID %q in the history in tick %d", contID, tickNo)
						s.Empty(resultHist.ContainerID)
					}
				}
				entityStore.RecordTick()
				if op, ok := tCase.operationAfterTick[tickNo]; ok {
					s.T().Logf("Exec operation=%s (tick %d). State after operation:", op, tickNo)
					switch op {
					case mapReset:
						entityStore.resetMaps()
					case deleteDeployment1:
						// purgeNoLock accepts deploymentID, not containerID
						entityStore.purgeNoLock("depl1")
					}
					s.T().Logf("\tHistorical container IDs (tick %d): %v", tickNo, prettyPrintHistoricalData(entityStore.historicalContainerIDs))
					s.T().Logf("\tAll container IDs (tick %d): %v", tickNo, maps.Keys(entityStore.containerIDMap))
				}
			}
		})
	}
}

func prettyPrintHistoricalData[M ~map[K1]map[K2]*entityStatus, K1 comparable, K2 comparable](data M) string {
	fragments := make([]string, 0, len(data))
	if len(data) == 0 {
		return "history is empty"
	}
	for ID, m := range data {
		for _, status := range m {
			fragments = append(fragments,
				fmt.Sprintf("[ID=%v, isHistorical=%t, ticksLeft=%d]", ID, status.isHistorical, status.ticksLeft))
		}
	}
	return strings.Join(fragments, "\n")
}

// endregion
