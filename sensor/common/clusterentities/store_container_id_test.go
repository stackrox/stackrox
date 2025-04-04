package clusterentities

func (s *ClusterEntitiesStoreTestSuite) TestMemoryAboutPastContainerIDs() {
	cases := map[string]struct {
		numTicksToRemember uint16
		entityUpdates      map[int][]eUpdate // tick -> updates
		// operationAfterTick defines tick IDs after which an operation should be simulated
		// (e.g., deletion of a container, or going offline).
		operationAfterTick    map[int]operation
		containerIDsAfterTick []map[string]whereThingIsStored
		publicIPsAtCleanup    []string
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1: no reset - should be in the map forever
				{"pod1": theMap},
			},
			publicIPsAtCleanup: []string{},
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// container deletion
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				{"pod1": nowhere}, // after tick 4: history expired - should be forgotten forever
				{"pod1": nowhere},
			},
		},
		"Re-adding normally deleted container (after history expired) should reset the history status": {
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
			containerIDsAfterTick: []map[string]whereThingIsStored{
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
		"Re-adding normally deleted container (while still in history) should reset the history status": {
			numTicksToRemember: 2,
			entityUpdates: map[int][]eUpdate{
				0: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
				4: {
					{
						deploymentID: "depl1",
						containerID:  "pod1",
						incremental:  true,
					},
				},
			},
			operationAfterTick: map[int]operation{1: deleteDeployment1},
			containerIDsAfterTick: []map[string]whereThingIsStored{
				{"pod1": theMap}, // before tick 1: container should be added immediately
				{"pod1": theMap}, // after tick 1
				// container deletion
				{"pod1": history}, // after tick 2: will remember that for one more tick
				{"pod1": history}, // after tick 3: will remember that for this last tick
				// add container again
				{"pod1": theMap}, // after tick 4 should be normally added to the map
				{"pod1": theMap},
				{"pod1": theMap},
			},
		},
	}
	for name, tCase := range cases {
		s.Run(name, func() {
			store := NewStoreWithMemory(tCase.numTicksToRemember, true)
			ipListener := newTestPublicIPsListener(s.T())
			store.RegisterPublicIPsListener(ipListener)
			// Set up the cleanup-assertions
			defer func() {
				s.True(store.UnregisterPublicIPsListener(ipListener))
				s.Equalf(len(tCase.publicIPsAtCleanup), ipListener.data.Cardinality(),
					"the listeners of public IPs have incorrect data at test cleanup")
				for gotIP := range ipListener.data {
					s.Contains(tCase.publicIPsAtCleanup, gotIP.AsNetIP().String())
				}
			}()

			for tickNo, expectation := range tCase.containerIDsAfterTick {
				// Add entities to the store (mimic data arriving from the K8s informers)
				if updatesForTick, ok := tCase.entityUpdates[tickNo]; ok {
					for _, update := range updatesForTick {
						store.Apply(map[string]*EntityData{
							update.deploymentID: entityUpdate("10.0.0.1", update.containerID, 80),
						}, update.incremental)
					}
				}
				// Assert on container IDs
				s.T().Logf("Container IDs (tick %d): %s", tickNo, store.containerIDsStore.String())
				for contID, whereFound := range expectation {
					result, found, _ := store.LookupByContainerID(contID)
					result2, found2, historical := store.containerIDsStore.lookupByContainer(contID)
					s.Equal(found2, found)
					s.Equal(result2, result)
					switch whereFound {
					case theMap:
						s.Truef(found, "expected to find contID %q in general in tick %d", contID, tickNo)
						s.Equalf(contID, result.ContainerID, "Expected the general result to have contID %q in tick %d. Result: %v", contID, tickNo, result)
						s.Falsef(historical, "expected contID %q to NOT be historical in tick %d", contID, tickNo)
					case history:
						s.Truef(found, "expected to find contID %q in general in tick %d", contID, tickNo)
						s.Equalf(contID, result.ContainerID, "Expected the general result to have contID %q in tick %d. Result: %v", contID, tickNo, result)
						s.Truef(historical, "expected contID %q to NOT be historical in tick %d", contID, tickNo)
					case nowhere:
						s.Falsef(found, "expected not to find contID %q at all in tick %d", contID, tickNo)
						s.Empty(result.ContainerID)
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
					s.T().Logf("\t\tContainer IDs (tick %d): %s", tickNo, store.containerIDsStore.String())

				}
			}
		})
	}
}
