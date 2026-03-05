package clusterentities

import (
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
)

func (s *ClusterEntitiesStoreTestSuite) TestMemoryAboutPastIPs() {
	cases := map[string]struct {
		numTicksToRemember uint16
		entityUpdates      map[int][]eUpdate // tick -> updates
		operationAfterTick map[int]operation
		endpointsAfterTick []map[string]whereThingIsStored
		publicIPsAtCleanup []string
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
						incremental:  false, // delete all data for the deployment, then apply update
					},
				},
			},
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"10.0.0.1": nowhere, "10.3.0.1": theMap}, // pre-tick 1: 10.0.0.1 should be overwritten immediately - only 10.3.0.1 should exist
				{"10.0.0.1": nowhere, "10.3.0.1": theMap}, // tick 1: only 10.3.0.1 should exist
				{"10.0.0.1": nowhere, "10.3.0.1": theMap}, // tick 2: only 10.3.0.1 should exist
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
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"10.0.0.1": history, "10.3.0.1": theMap}, // pre-tick 1: both must exist
				{"10.0.0.1": nowhere, "10.3.0.1": theMap}, // after-tick 1: only 10.3.0.1 should exist
				{"10.0.0.1": nowhere, "10.3.0.1": theMap}, // after-tick 2: only 10.3.0.1 should exist
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
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"10.0.0.1": theMap}, // tick 0: update0
				{"10.0.0.1": theMap}, // tick 1: mark update0 as historical
				{"10.0.0.1": theMap}, // tick 2: historical update0 exists; add again in update2
				{"10.0.0.1": theMap}, // tick 3: historical update0 would be deleted, but update2 shall exist
				{"10.0.0.1": theMap}, // tick 4: update2 must exist
				{"10.0.0.1": theMap}, // tick 5: update2 must exist
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
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"10.0.0.1": history, "10.3.0.1": theMap}, // pre-tick 1: first IP is immediately changed and goes to history
				{"10.0.0.1": history, "10.3.0.1": theMap}, // after-tick 1: same after one tick
				{"10.0.0.1": nowhere, "10.3.0.1": theMap}, // after-tick 2: only 10.3.0.1 should exist
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
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"10.0.0.1": history, "10.3.0.1": theMap, "20.0.0.1": theMap, "20.3.0.1": theMap},
				{"10.0.0.1": history, "10.3.0.1": theMap, "20.0.0.1": theMap, "20.3.0.1": theMap},
				{"10.0.0.1": nowhere, "10.3.0.1": theMap, "20.0.0.1": theMap, "20.3.0.1": theMap},
			},
			publicIPsAtCleanup: []string{"20.0.0.1", "20.3.0.1"},
		},
		"Public IPs should be gone on the 2nd tick": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "34.118.224.226",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "34.118.224.227",
						port:         80,
						incremental:  false,
					},
				},
			},
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"34.118.224.226": history, "34.118.224.227": theMap}, // pre-tick 1: first IP is immediately changed and goes to history
				{"34.118.224.226": history, "34.118.224.227": theMap}, // after-tick 1: same after one tick
				{"34.118.224.226": nowhere, "34.118.224.227": theMap}, // after-tick 2: only 34.118.224.227 should exist
			},
			publicIPsAtCleanup: []string{"34.118.224.227"},
		},
		"Memory disabled with reset after tick 1 should make IPs be forgotten before tick 2": {
			numTicksToRemember: 0,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "34.118.224.226",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "34.118.224.227",
						port:         80,
						incremental:  false,
					},
				},
			},
			operationAfterTick: map[int]operation{1: mapReset},
			endpointsAfterTick: []map[string]whereThingIsStored{
				{"34.118.224.226": nowhere, "34.118.224.227": theMap}, // pre-tick 1:
				// the first IP is immediately forgotten with history disabled
				{"34.118.224.226": nowhere, "34.118.224.227": theMap},  // after-tick 1: before reset all should be the same
				{"34.118.224.226": nowhere, "34.118.224.227": nowhere}, // after-tick 2: after map reset all data is gone
			},
			publicIPsAtCleanup: []string{},
		},
		"Memory enabled with reset after tick 1 should make historical IPs be remembered": {
			numTicksToRemember: 3,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "34.118.224.226",
						port:         80,
						incremental:  true,
					},
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "34.118.224.227",
						port:         80,
						incremental:  false,
					},
				},
			},
			operationAfterTick: map[int]operation{1: mapReset},
			endpointsAfterTick: []map[string]whereThingIsStored{
				// pre-tick 1: the first IP is immediately overwritten and put into history
				{"34.118.224.226": history, "34.118.224.227": theMap},
				// after-tick 1, but before the reset - no changes expected
				{"34.118.224.226": history, "34.118.224.227": theMap},
				// reset happens here
				// after-tick 2: IP .226 shall be remembered for one more tick, whereas 227 is freshly added to history
				{"34.118.224.226": history, "34.118.224.227": history},
				// after-tick 3: IP .226 expired after 3 ticks
				{"34.118.224.226": nowhere, "34.118.224.227": history},
				{"34.118.224.226": nowhere, "34.118.224.227": history},
				// after-tick 5: IP .227 expired after being in the history for 3 ticks
				{"34.118.224.226": nowhere, "34.118.224.227": nowhere},
			},
			publicIPsAtCleanup: []string{},
		},
		"Memory enabled with IP-overwrite in tick 1 and deleting unknown deployment in tick 2": {
			numTicksToRemember: 3,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl-known",
						containerID:  "pod1",
						ipAddr:       "34.118.224.226",
						port:         80,
						incremental:  true,
					},
				},
				1: {
					{
						deploymentID: "depl-known",
						containerID:  "pod1",
						ipAddr:       "34.118.224.227",
						port:         80,
						incremental:  false,
					},
				},
			},
			operationAfterTick: map[int]operation{2: deleteDeployment1}, // depl1 is not known to the store
			endpointsAfterTick: []map[string]whereThingIsStored{
				// pre-tick 1: the IP 226 is added
				{"34.118.224.226": theMap, "34.118.224.227": nowhere},
				// after-tick 1: IP 227 overwrites the 226
				{"34.118.224.226": history, "34.118.224.227": theMap},
				// after-tick 2: same as after 1
				{"34.118.224.226": history, "34.118.224.227": theMap},
				// deletion of not-existing deployment happens here!
				// after-tick 3: same as after tick 2
				{"34.118.224.226": history, "34.118.224.227": theMap},
				// after-tick 4: IP 226 expires after being in history for 3 ticks
				{"34.118.224.226": nowhere, "34.118.224.227": theMap},
				// after-tick 5: and it goes like this forever
				{"34.118.224.226": nowhere, "34.118.224.227": theMap},
			},
			publicIPsAtCleanup: []string{"34.118.224.227"},
		},
		"One IP belongs to multiple deployments, memory 1": {
			numTicksToRemember: 1,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						ipAddr:       "20.0.0.1",
						port:         80,
						incremental:  true,
					},
				},
				1: {
					{
						deploymentID: "depl2",
						containerID:  "pod2",
						ipAddr:       "20.0.0.1",
						port:         82,
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{2: deleteDeployment1},
			endpointsAfterTick: []map[string]whereThingIsStored{
				// pre-tick 1: depl1 is added
				{"20.0.0.1": theMap},
				// after-tick 1: depl2 is added
				{"20.0.0.1": theMap},
				// after-tick 2: no change here
				{"20.0.0.1": theMap},
				// deleting depl1
				// after-tick 3: the IP still belongs to depl2,
				// so it should be in current, while the entry for depl1 should be in history
				{"20.0.0.1": inBoth},
				// The historical IP of depl1 expires now
				// after-tick 4
				{"20.0.0.1": theMap},
				// after-tick 5
				{"20.0.0.1": theMap},
			},
			publicIPsAtCleanup: []string{"20.0.0.1"},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			store := NewStore(tCase.numTicksToRemember, nil, true)
			ipListener := newTestPublicIPsListener(s.T())
			store.RegisterPublicIPsListener(ipListener)
			// Set up the cleanup-assertions
			defer func() {
				s.Equalf(len(tCase.publicIPsAtCleanup), ipListener.data.Cardinality(),
					"the listeners of public IPs have incorrect data at test cleanup")
				for gotIP := range ipListener.data {
					s.Containsf(tCase.publicIPsAtCleanup, gotIP.AsNetIP().String(), "unexpected IP %s in the ipListener", gotIP)
				}
				s.True(store.UnregisterPublicIPsListener(ipListener))
			}()

			for tickNo, expect := range tCase.endpointsAfterTick {
				// Add entities to the store (mimic data arriving from the K8s informers)
				if updatesForTick, ok := tCase.entityUpdates[tickNo]; ok {
					s.T().Logf("Applying %d updates for tick %d", len(updatesForTick), tickNo)
					for _, update := range updatesForTick {
						store.Apply(map[string]*EntityData{
							update.deploymentID: entityUpdate(update.ipAddr, update.containerID, update.port),
						}, update.incremental)
					}
				}
				// Assert on IPs
				s.T().Logf("IPs (tick %d):\n%s", tickNo, store.podIPsStore.String())
				s.T().Logf("IP listener (tick %d): %s", tickNo, ipListener.String())
				// convert to slice of strings to enable using Contains assertion
				var historicalIPs []string
				for address := range store.podIPsStore.historicalIPs {
					historicalIPs = append(historicalIPs, address.String())
				}
				var currentIPs []string
				for address := range store.podIPsStore.ipMap {
					currentIPs = append(currentIPs, address.String())
				}
				for endpointIP, whereFound := range expect {
					netIP := net.ParseIP(endpointIP)
					current, historical := store.podIPsStore.LookupByNetAddr(netIP, 80)
					switch whereFound {
					case theMap:
						s.Greaterf(len(current), 0, "IP address lookup should return at least one result from the map")
						s.Containsf(currentIPs, endpointIP, "expected IP %s to be found in the map in tick %d", endpointIP, tickNo)
						s.NotContainsf(historicalIPs, endpointIP, "expected IP %s to be absent in history in tick %d", endpointIP, tickNo)
					case history:
						s.Greaterf(len(historical), 0, "IP address lookup should return at least one result from the history")
						s.NotContainsf(currentIPs, endpointIP, "expected IP %s to be absent in the map in tick %d", endpointIP, tickNo)
						s.Containsf(historicalIPs, endpointIP, "expected IP %s to be found in history in tick %d", endpointIP, tickNo)
					case inBoth:
						s.Greaterf(len(current), 0, "IP address lookup should return at least one result from the map")
						s.Greaterf(len(historical), 0, "IP address lookup should return at least one result from the history")
						s.Containsf(currentIPs, endpointIP, "expected IP %s to be found in the map in tick %d", endpointIP, tickNo)
						s.Containsf(historicalIPs, endpointIP, "expected IP %s to be found in history in tick %d", endpointIP, tickNo)
					case nowhere:
						s.Lenf(current, 0, "IP address lookup should return empty result")
						s.Lenf(historical, 0, "IP address lookup should return empty result")
						s.NotContainsf(currentIPs, endpointIP, "expected IP %s to be absent in the map in tick %d", endpointIP, tickNo)
						s.NotContainsf(historicalIPs, endpointIP, "expected IP %s to be absent in history in tick %d", endpointIP, tickNo)
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
					s.T().Logf("\t\tIPs (tick %d): %s", tickNo, store.podIPsStore.String())

				}
			}

		})
	}
}

// totalDeploymentsInIPMap returns the total number of deployment IDs referenced
// across all entries in ipMap (counts duplicates across different IPs).
func totalDeploymentsInIPMap(store *podIPsStore) int {
	count := 0
	for _, deplSet := range store.ipMap {
		count += deplSet.Cardinality()
	}
	return count
}

// totalIPsInReverseIPMap returns the total number of IP addresses referenced
// across all entries in reverseIPMap.
func totalIPsInReverseIPMap(store *podIPsStore) int {
	count := 0
	for _, ipSet := range store.reverseIPMap {
		count += ipSet.Cardinality()
	}
	return count
}

// TestIPMapConsistencyAfterIPRecyclingViaApply simulates the IP recycling scenario that occurs when Sensor processes
// events out of order (new pod's IP update arrives before old pod's removal)
// It reproduces the same issue as in TestDeleteDeploymentFromCurrentRemovesStaleEntries,
// but uses different API (Store.Apply instead of deleteDeploymentFromCurrent).
//
// The order of event arrival is crucial for reproducing this issue: the bug
// only triggers when the new deployment's update is processed while the old
// deployment's stale IP is still in ipMap (cardinality >= 2). When events
// arrive in order (old deployment cleaned up first), cardinality is 1 at
// deletion time and cleanup works correctly.
//
// This out-of-order pattern has been observed in production on clusters with
// high pod churn, where Kubernetes informer events for different deployments
// are delivered asynchronously and the recycled-IP event arrives before the
// old pod's termination is fully processed.
func (s *ClusterEntitiesStoreTestSuite) TestIPMapConsistencyAfterIPRecyclingViaApply() {
	s.Run("IP recycled between two deployments leaves no leftovers after deletion", func() {
		store := NewStore(0, nil, true)

		ip := net.ParseIP("10.0.0.1")

		// deplA gets pod with IP 10.0.0.1.
		store.Apply(map[string]*EntityData{
			"deplA": entityUpdate("10.0.0.1", "containerA", 80),
		}, false)

		s.T().Logf("After deplA appears with IP 10.0.0.1:\n%s", store.podIPsStore.String())
		s.Equal(1, len(store.podIPsStore.ipMap), "ipMap should have 1 IP")
		s.Equal(1, len(store.podIPsStore.reverseIPMap), "reverseIPMap should have 1 deployment")

		// K8s recycles IP 10.0.0.1 to deplB.
		// Sensor processes deplB's update BEFORE processing deplA's removal.
		store.Apply(map[string]*EntityData{
			"deplB": entityUpdate("10.0.0.1", "containerB", 80),
		}, false)

		s.T().Logf("After deplB is added and recycles IP 10.0.0.1 (deplA not yet cleaned up):\n%s", store.podIPsStore.String())
		s.Equal(2, store.podIPsStore.ipMap[ip].Cardinality(),
			"ipMap should have 2 deployments for the recycled IP")

		// deplA is updated — its old pod is gone, new pod has IP 10.0.0.2.
		store.Apply(map[string]*EntityData{
			"deplA": entityUpdate("10.0.0.2", "containerA2", 80),
		}, false)

		s.T().Logf("After deplA moves to 10.0.0.2 (should release IP 10.0.0.1):\n%s", store.podIPsStore.String())
		s.False(store.podIPsStore.ipMap[ip].Contains("deplA"),
			"deplA should be absent from ipMap[10.0.0.1] after moving to a new IP")

		// Both deployments are deleted from K8s.
		store.Apply(map[string]*EntityData{"deplA": {}}, false)
		store.Apply(map[string]*EntityData{"deplB": {}}, false)

		s.T().Logf("After both deplA and deplB are deleted:\n%s", store.podIPsStore.String())
		s.Empty(store.podIPsStore.ipMap,
			"ipMap should be empty after all deployments are deleted")
		s.Empty(store.podIPsStore.reverseIPMap,
			"reverseIPMap should be empty after all deployments are deleted")
	})

	s.Run("repeated IP recycling accumulates stale entries in ipMap", func() {
		store := NewStore(0, nil, true)

		// Simulate 5 successive deployments each getting IP 10.0.0.1.
		// Each new deployment arrives before the previous one is cleaned up,
		// mimicking rapid IP recycling in a busy cluster.
		deployments := []string{"deplA", "deplB", "deplC", "deplD", "deplE"}
		for i, deplID := range deployments {
			store.Apply(map[string]*EntityData{
				deplID: entityUpdate("10.0.0.1", "container-"+deplID, 80),
			}, false)
			s.T().Logf("After %s gets 10.0.0.1 (%d/%d): ipMap cardinality=%d, reverseIPMap entries=%d",
				deplID, i+1, len(deployments),
				store.podIPsStore.ipMap[net.ParseIP("10.0.0.1")].Cardinality(),
				len(store.podIPsStore.reverseIPMap))
		}

		s.T().Logf("State after all 5 deployments added:\n%s", store.podIPsStore.String())

		// Delete all deployments from K8s.
		for _, deplID := range deployments {
			store.Apply(map[string]*EntityData{deplID: {}}, false)
			s.T().Logf("After deleting %s: ipMap deployment refs=%d, reverseIPMap entries=%d",
				deplID,
				totalDeploymentsInIPMap(store.podIPsStore),
				len(store.podIPsStore.reverseIPMap))
		}

		s.T().Logf("Final state:\n%s", store.podIPsStore.String())

		// The `ipMap` and `reverseIPMap` must agree: both should be empty.
		s.Equal(0, totalIPsInReverseIPMap(store.podIPsStore),
			"reverseIPMap should have 0 IP references after all deployments are deleted")
		s.Equal(0, totalDeploymentsInIPMap(store.podIPsStore),
			"ipMap should have 0 deployment references after all deployments are deleted (stale entries indicate a bug)")
		s.Empty(store.podIPsStore.ipMap,
			"ipMap should be empty after all deployments are deleted")
	})
}

func ptr[T any](v T) *T { return &v }

// This is a different version of `TestIPMapConsistencyAfterIPRecyclingViaApply`
// that uses direct calls to `deleteDeploymentFromCurrent` instead of `Store.Apply`.
//
// When a deployment is deleted from `ipMap` while sharing an IP with another
// deployment, its entry must be removed from `ipMap` to keep `ipMap` and
// `reverseIPMap` consistent.
func (s *ClusterEntitiesStoreTestSuite) TestDeleteDeploymentFromCurrentRemovesStaleEntries() {
	ip := net.ParseIP("10.0.0.1")
	makeData := func(ipStr string) EntityData {
		ed := EntityData{}
		ed.AddIP(net.ParseIP(ipStr))
		return ed
	}

	s.Run("deleting one of two deployments sharing an IP cleans ipMap", func() {
		store := newPodIPsStoreWithMemory(0)

		store.applyNoLock(map[string]*EntityData{"deplA": ptr(makeData("10.0.0.1"))}, true)
		store.applyNoLock(map[string]*EntityData{"deplB": ptr(makeData("10.0.0.1"))}, true)

		// Precondition: both deployments share the IP.
		s.Require().Equal(2, store.ipMap[ip].Cardinality())
		s.Require().True(store.ipMap[ip].Contains("deplA"))
		s.Require().True(store.ipMap[ip].Contains("deplB"))

		store.deleteDeploymentFromCurrent("deplA")

		s.False(store.ipMap[ip].Contains("deplA"),
			"deplA should be absent from ipMap after deleteDeploymentFromCurrent")
		s.True(store.ipMap[ip].Contains("deplB"),
			"deplB should still be in ipMap after deleteDeploymentFromCurrent")
		s.Equal(1, store.ipMap[ip].Cardinality(),
			"ipMap entry should have exactly 1 deployment left")
		s.Empty(store.reverseIPMap["deplA"],
			"reverseIPMap should not reference the deleted deployment")
	})

	s.Run("deleting all deployments that shared an IP leaves maps empty", func() {
		store := newPodIPsStoreWithMemory(0)

		store.applyNoLock(map[string]*EntityData{"deplA": ptr(makeData("10.0.0.1"))}, true)
		store.applyNoLock(map[string]*EntityData{"deplB": ptr(makeData("10.0.0.1"))}, true)

		store.deleteDeploymentFromCurrent("deplA")
		store.deleteDeploymentFromCurrent("deplB")

		s.Empty(store.ipMap,
			"ipMap should be empty after all deployments sharing the IP are deleted")
		s.Empty(store.reverseIPMap,
			"reverseIPMap should be empty after all deployments are deleted")
	})

	s.Run("ipMap grows with stale entries on repeated IP recycling", func() {
		store := newPodIPsStoreWithMemory(0)

		// Simulate IP recycling: deplA gets the IP, then deplB, then deplC.
		store.applyNoLock(map[string]*EntityData{"deplA": ptr(makeData("10.0.0.1"))}, true)
		store.applyNoLock(map[string]*EntityData{"deplB": ptr(makeData("10.0.0.1"))}, true)
		store.deleteDeploymentFromCurrent("deplA")

		store.applyNoLock(map[string]*EntityData{"deplC": ptr(makeData("10.0.0.1"))}, true)
		store.deleteDeploymentFromCurrent("deplB")

		store.deleteDeploymentFromCurrent("deplC")

		s.Equal(0, totalDeploymentsInIPMap(store),
			"ipMap should have 0 deployment references after all deployments are deleted")
		s.Equal(0, totalIPsInReverseIPMap(store),
			"reverseIPMap should have 0 IP references after all deployments are deleted")
		s.Empty(store.ipMap,
			"ipMap should be empty after all deployments are deleted")
	})
}

func (s *ClusterEntitiesStoreTestSuite) TestChangingIPsAndExternalEntities() {
	entityStore := NewStore(0, nil, false)
	type expectation struct {
		ip           string
		port         uint16
		deploymentID string
	}
	type eUpdate struct {
		deploymentID string
		ipAddr       string
		port         uint16
		incremental  bool
	}
	cases := map[string]struct {
		entityUpdates     []eUpdate
		expectedEndpoints []expectation
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
			expectedEndpoints: []expectation{
				{
					ip:           "10.0.0.1",
					port:         80,
					deploymentID: "pod1",
				},
				{
					ip:           "10.3.0.1",
					port:         80,
					deploymentID: "pod1",
				},
			},
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
			expectedEndpoints: []expectation{
				{
					ip:           "10.3.0.1",
					port:         80,
					deploymentID: "pod1",
				},
				{
					ip:           "10.0.0.2",
					port:         80,
					deploymentID: "pod2",
				},
			},
		},
		"Lookup by NetAddr finds data when endpoint cannot be found": {
			entityUpdates: []eUpdate{
				{
					deploymentID: "pod2",
					ipAddr:       "20.0.0.2",
					port:         99,
					incremental:  false,
				},
			},
			// We will not find endpoint for port 80, but thanks to the ipLookup,
			// should still be able to find pod2.
			expectedEndpoints: []expectation{
				{
					ip:           "20.0.0.2",
					port:         80,
					deploymentID: "pod2",
				},
			},
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
				result := entityStore.LookupByEndpoint(buildEndpoint(expectedEndpoint.ip, expectedEndpoint.port))
				s.Require().Lenf(result, 1, "Expected endpoint %q not found", expectedEndpoint)
				s.Equal(networkgraph.EntityForDeployment(expectedEndpoint.deploymentID), result[0].Entity)
				s.Require().Len(result[0].ContainerPorts, 1)
				s.Equal(expectedEndpoint.port, result[0].ContainerPorts[0])
			}
		})
	}
}
