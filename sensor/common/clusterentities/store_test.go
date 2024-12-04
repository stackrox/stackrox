package clusterentities

import (
	"testing"

	"github.com/stretchr/testify/suite"
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
	ipAddr       string
	port         uint16
	portName     string
	incremental  bool
}

type whereThingIsStored string

const (
	// the thing will be found in history
	history whereThingIsStored = "history"
	// the thing will be found in the current map
	theMap whereThingIsStored = "the-map"
	// the thing will be found in the current map and in the history
	inBoth whereThingIsStored = "in-both"
	// the thing will not be found
	nowhere whereThingIsStored = "nowhere"
)

func (s *ClusterEntitiesStoreTestSuite) TestMemoryWhenGoingOffline() {
	cases := map[string]struct {
		numTicksToRemember     uint16
		initialState           map[string]*EntityData
		wantMapSizeOnline      int
		wantHistorySizeOnline  int
		wantMapSizeOffline     int
		wantHistorySizeOffline int
	}{
		"Going offline with memory enabled should preserve history": {
			numTicksToRemember: 1,
			initialState: map[string]*EntityData{
				"depl1": entityUpdate("10.0.0.1", "container1", 80),
				"depl2": entityUpdate("10.0.0.2", "container2", 8080),
			},
			wantMapSizeOnline:      2,
			wantHistorySizeOnline:  0,
			wantMapSizeOffline:     0,
			wantHistorySizeOffline: 2,
		},
		"Going offline with memory disabled should purge entire history": {
			numTicksToRemember: 0,
			initialState: map[string]*EntityData{
				"depl1": entityUpdate("10.0.0.1", "container1", 80),
				"depl2": entityUpdate("10.0.0.2", "container2", 8080),
			},
			wantMapSizeOnline:      2,
			wantHistorySizeOnline:  0,
			wantMapSizeOffline:     0,
			wantHistorySizeOffline: 0,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			entityStore := NewStoreWithMemory(tc.numTicksToRemember, true)
			entityStore.Apply(tc.initialState, true)
			// We start online
			s.Len(entityStore.podIPsStore.ipMap, tc.wantMapSizeOnline)
			s.Len(entityStore.endpointsStore.endpointMap, tc.wantMapSizeOnline)
			s.Len(entityStore.containerIDsStore.containerIDMap, tc.wantMapSizeOnline)

			s.Len(entityStore.podIPsStore.historicalIPs, tc.wantHistorySizeOnline)
			s.Len(entityStore.endpointsStore.reverseHistoricalEndpoints, tc.wantHistorySizeOnline)
			s.Len(entityStore.containerIDsStore.historicalContainerIDs, tc.wantHistorySizeOnline)

			s.T().Logf("%s", string(entityStore.Debug()))

			// Transition to offline
			entityStore.Cleanup()

			s.Len(entityStore.podIPsStore.ipMap, tc.wantMapSizeOffline, "error in current IPs after cleanup")
			s.Len(entityStore.endpointsStore.endpointMap, tc.wantMapSizeOffline, "error in current endpoints after cleanup")
			s.Len(entityStore.containerIDsStore.containerIDMap, tc.wantMapSizeOffline, "error in current container IDs after cleanup")

			s.Len(entityStore.podIPsStore.historicalIPs, tc.wantHistorySizeOffline, "error in historical IPs after cleanup")
			s.Len(entityStore.endpointsStore.historicalEndpoints, tc.wantHistorySizeOffline, "error in historical endpoints after cleanup")
			s.Len(entityStore.containerIDsStore.historicalContainerIDs, tc.wantHistorySizeOffline, "error in historical container IDs after cleanup")
		})
	}
}
