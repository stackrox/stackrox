package clusterentities

import (
	"slices"
	"sort"
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/assert"
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
			entityStore := NewStore(tc.numTicksToRemember, nil, true)
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

func TestEntityData_GetContainerIDs(t *testing.T) {
	tests := map[string]struct {
		edFun              func() *EntityData
		containerNameQuery string
		wantContainerIDs   []string
	}{
		"Single containerID with matching name": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddContainerID("abc", ContainerMetadata{
					ContainerName: "container-abc",
				})
				return ed
			},
			containerNameQuery: "container-abc",
			wantContainerIDs:   []string{"abc"},
		},
		"Single containerID with no match in name": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddContainerID("abc", ContainerMetadata{
					ContainerName: "container-123",
				})
				return ed
			},
			containerNameQuery: "container-abc",
			wantContainerIDs:   []string{},
		},
		"Multiple containers sorted by name": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddContainerID("abc", ContainerMetadata{
					ContainerName: "container-abc",
				})
				ed.AddContainerID("def", ContainerMetadata{
					ContainerName: "container-def",
				})
				return ed
			},
			containerNameQuery: "container-abc",
			wantContainerIDs:   []string{"abc"},
		},
		"Multiple containers unsorted by name": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddContainerID("def", ContainerMetadata{
					ContainerName: "container-def",
				})
				ed.AddContainerID("abc", ContainerMetadata{
					ContainerName: "container-abc",
				})
				return ed
			},
			containerNameQuery: "container-abc",
			wantContainerIDs:   []string{"abc"},
		},
		"Multiple container IDs for the same container name (impossible in prod)": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddContainerID("def", ContainerMetadata{
					ContainerName: "container-def",
				})
				ed.AddContainerID("xyz", ContainerMetadata{
					ContainerName: "container-def",
				})
				return ed
			},
			containerNameQuery: "container-def",
			wantContainerIDs:   []string{"def", "xyz"},
		},
		"No Container ID": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				return ed
			},
			containerNameQuery: "container-abc",
			wantContainerIDs:   []string{},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ed := tt.edFun()
			gotContainerIDs := ed.GetContainerIDs(tt.containerNameQuery)
			// Sort as GetDetails is not guaranteed to return sorted data.
			slices.Sort(gotContainerIDs)
			assert.ElementsMatch(t, tt.wantContainerIDs, gotContainerIDs)
		})
	}
}

func TestEntityData_GetPodIPs(t *testing.T) {
	tests := map[string]struct {
		edFun      func() *EntityData
		wantPodIPs []net.IPAddress
	}{
		"Single values": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddIP(net.ParseIP("10.0.0.1"))
				return ed
			},
			wantPodIPs: []net.IPAddress{net.ParseIP("10.0.0.1")},
		},
		"Multiple sorted values": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddIP(net.ParseIP("10.0.0.1"))
				ed.AddIP(net.ParseIP("10.0.0.2"))
				return ed
			},
			wantPodIPs: []net.IPAddress{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2")},
		},
		"Multiple unsorted values": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddIP(net.ParseIP("10.0.0.9"))
				ed.AddIP(net.ParseIP("10.0.0.2"))
				return ed
			},
			wantPodIPs: []net.IPAddress{net.ParseIP("10.0.0.9"), net.ParseIP("10.0.0.2")},
		},
		"Invalid IP": {
			edFun: func() *EntityData {
				ed := &EntityData{}
				ed.AddIP(net.ParseIP("foo.bar.baz.boom"))
				ed.AddIP(net.ParseIP("10.0.0.2"))
				ed.AddContainerID("abc", ContainerMetadata{})
				return ed
			},
			wantPodIPs: []net.IPAddress{net.ParseIP("10.0.0.2")},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ed := tt.edFun()
			gotPodIPs := ed.GetValidIPs()
			// Sort as GetDetails is not guaranteed to return sorted data.
			sort.Slice(gotPodIPs, func(i, j int) bool {
				return net.IPAddressLess(gotPodIPs[i], gotPodIPs[j])
			})
			assert.ElementsMatch(t, tt.wantPodIPs, gotPodIPs)
		})
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestPublicIPListenerConditionalUpdate() {
	tests := map[string]struct {
		setup    func(store *Store)
		action   func(store *Store)
		expected []string
	}{
		"private-only incremental add should not trigger listener": {
			action: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("10.0.0.1", "cont1", 80),
				}, true)
			},
			expected: nil,
		},
		"incremental delete-only should not trigger listener": {
			setup: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("10.0.0.1", "cont1", 80),
				}, true)
			},
			action: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("", "", 0),
				}, true)
			},
			expected: nil,
		},
		"public IP incremental add should trigger listener": {
			action: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("34.118.224.226", "cont1", 80),
				}, true)
			},
			expected: []string{"34.118.224.226"},
		},
		"public IP non-incremental replace should trigger listener": {
			setup: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("8.8.8.8", "cont1", 80),
				}, true)
			},
			action: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("1.1.1.1", "cont2", 80),
				}, false)
			},
			// 8.8.8.8 moved to history, 1.1.1.1 added to current
			expected: []string{"1.1.1.1", "8.8.8.8"},
		},
		"public IP removal via delete should trigger listener": {
			setup: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("8.8.8.8", "cont1", 80),
				}, true)
			},
			action: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("", "", 0),
				}, false)
			},
			// 8.8.8.8 moved to history (memorySize=1), still present
			expected: []string{"8.8.8.8"},
		},
		"mixed private and public should report only public": {
			action: func(store *Store) {
				ed := &EntityData{}
				ep1 := buildEndpoint("10.0.0.1", 80)
				ed.AddEndpoint(ep1, EndpointTargetInfo{ContainerPort: 80})
				ed.AddIP(ep1.IPAndPort.Address)
				ep2 := buildEndpoint("8.8.8.8", 443)
				ed.AddEndpoint(ep2, EndpointTargetInfo{ContainerPort: 443})
				ed.AddIP(ep2.IPAndPort.Address)
				store.Apply(map[string]*EntityData{"depl1": ed}, true)
			},
			expected: []string{"8.8.8.8"},
		},
		"replacing public with private should still show history": {
			setup: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("8.8.8.8", "cont1", 80),
				}, true)
			},
			action: func(store *Store) {
				store.Apply(map[string]*EntityData{
					"depl1": entityUpdate("10.0.0.1", "cont2", 80),
				}, false)
			},
			// 8.8.8.8 moved to history (memorySize=1), still present
			expected: []string{"8.8.8.8"},
		},
		"cross-store: updating one sub-store must not drop IPs from the other": {
			setup: func(store *Store) {
				ed := &EntityData{}
				ed.AddEndpoint(buildEndpoint("8.8.8.8", 80), EndpointTargetInfo{ContainerPort: 80})
				store.Apply(map[string]*EntityData{"depl1": ed}, true)
			},
			action: func(store *Store) {
				ed := &EntityData{}
				ed.AddIP(net.ParseIP("1.1.1.1"))
				store.Apply(map[string]*EntityData{"depl2": ed}, true)
			},
			expected: []string{"8.8.8.8", "1.1.1.1"},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			store := NewStore(1, nil, false)
			listener := newTestPublicIPsListener(s.T())
			store.RegisterPublicIPsListener(listener)
			defer store.UnregisterPublicIPsListener(listener)

			if tc.setup != nil {
				tc.setup(store)
			}
			tc.action(store)

			if tc.expected == nil {
				s.Empty(listener.data, "expected no public IPs")
			} else {
				s.Equal(len(tc.expected), listener.data.Cardinality(), "wrong number of public IPs")
				for _, ip := range tc.expected {
					s.True(listener.data.Contains(net.ParseIP(ip)), "missing expected IP %s", ip)
				}
			}
		})
	}
}

func (s *ClusterEntitiesStoreTestSuite) TestPublicIPReaddAfterRemoval() {
	store := NewStore(1, nil, false)
	listener := newTestPublicIPsListener(s.T())
	store.RegisterPublicIPsListener(listener)
	defer store.UnregisterPublicIPsListener(listener)

	// Add a public IP, then remove it via replacement, then expire history.
	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("8.8.8.8", "cont1", 80),
	}, true)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")))

	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("10.0.0.1", "cont2", 80),
	}, false)
	store.RecordTick()
	s.Empty(listener.data, "public IP should be gone after history expiry")

	// Re-add the same public IP — ref count must resurrect from zero.
	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("8.8.8.8", "cont3", 80),
	}, false)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")), "public IP should reappear after re-add")
}

func (s *ClusterEntitiesStoreTestSuite) TestPublicIPSharedBetweenDeployments() {
	store := NewStore(1, nil, false)
	listener := newTestPublicIPsListener(s.T())
	store.RegisterPublicIPsListener(listener)
	defer store.UnregisterPublicIPsListener(listener)

	// Two deployments share the same public IP (different ports).
	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("8.8.8.8", "cont1", 80),
	}, true)
	store.Apply(map[string]*EntityData{
		"depl2": entityUpdate("8.8.8.8", "cont2", 443),
	}, true)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")))

	// Remove one deployment — the IP should survive via the other's ref count.
	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("", "", 0),
	}, false)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")), "public IP should persist while second deployment still uses it")

	// Remove the second deployment — IP moves to history.
	store.Apply(map[string]*EntityData{
		"depl2": entityUpdate("", "", 0),
	}, false)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")), "public IP should be in history")

	// Expire history.
	store.RecordTick()
	s.Empty(listener.data, "public IP should be gone after history expiry")
}

func (s *ClusterEntitiesStoreTestSuite) TestPublicIPHistoryExpiry() {
	store := NewStore(1, nil, false)
	listener := newTestPublicIPsListener(s.T())
	store.RegisterPublicIPsListener(listener)
	defer store.UnregisterPublicIPsListener(listener)

	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("8.8.8.8", "cont1", 80),
	}, true)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")))

	// Replace with private — public IP moves to history
	store.Apply(map[string]*EntityData{
		"depl1": entityUpdate("10.0.0.1", "cont2", 80),
	}, false)
	s.True(listener.data.Contains(net.ParseIP("8.8.8.8")), "should still be in history")

	// Tick expires history (memorySize=1 means 1 tick to expire)
	store.RecordTick()
	s.Empty(listener.data, "history should have expired after tick")
}
