package clusterentities

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/heritage"
	"github.com/stretchr/testify/assert"
)

// mockHeritageManager implements HeritageManager for testing
type mockHeritageManager struct {
	data               []*heritage.SensorMetadata
	currentPodIP       string
	currentContainerID string
	setCalled          bool
	isEnabled          bool
}

func (m *mockHeritageManager) IsEnabled() bool {
	return m.isEnabled
}

func (m *mockHeritageManager) GetData(ctx context.Context) []*heritage.SensorMetadata {
	return m.data
}

func (m *mockHeritageManager) SetCurrentSensorData(podIP, containerID string) {
	m.currentPodIP = podIP
	m.currentContainerID = containerID
	m.setCalled = true
}

func TestStore_ApplyHeritageDataOnce(t *testing.T) {
	tests := map[string]struct {
		setupPastData        []*heritage.SensorMetadata
		setupCurrentMetadata bool
		featEnabled          bool
		expectedSignalDone   bool
	}{
		"should signal done when heritage data applied successfully once": {
			setupPastData: []*heritage.SensorMetadata{
				{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
			},
			setupCurrentMetadata: true,
			featEnabled:          true,
			expectedSignalDone:   true,
		},
		"should not signal done when no heritage data available": {
			setupPastData:        []*heritage.SensorMetadata{},
			setupCurrentMetadata: true,
			featEnabled:          true,
			expectedSignalDone:   false,
		},
		"should not signal done when missing current sensor metadata": {
			setupPastData: []*heritage.SensorMetadata{
				{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
			},
			setupCurrentMetadata: false,
			featEnabled:          true,
			expectedSignalDone:   false,
		},
		"should signal when feature disabled": {
			setupPastData:        []*heritage.SensorMetadata{},
			setupCurrentMetadata: true,
			featEnabled:          false,
			expectedSignalDone:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockHM := &mockHeritageManager{data: tt.setupPastData, isEnabled: tt.featEnabled}
			store := NewStore(0, mockHM, true)

			// Setup current sensor metadata if required
			if tt.setupCurrentMetadata {
				currentData := createSensorEntityData("current123", "10.2.2.2")
				store.RememberCurrentSensorMetadata("sensor-deploy-1", currentData)
			}

			// Call multiple times to verify single execution
			store.ApplyDataFromHeritageOnce()
			store.ApplyDataFromHeritageOnce()
			store.ApplyDataFromHeritageOnce()

			// Verify signal state
			assert.Equal(t, tt.expectedSignalDone, store.heritageApplied.IsDone())
		})
	}
}

func TestStore_applyHeritageData(t *testing.T) {
	const deployID = "sensor-deploy-1"
	tests := map[string]struct {
		currentDeployID   string
		currentEntityData *EntityData
		want              bool
	}{
		"should return true when current metadata is available": {
			currentDeployID:   deployID,
			currentEntityData: createSensorEntityData("current123", "10.2.2.2"),
			want:              true,
		},
		"should return false when current entity data is missing": {
			currentDeployID:   deployID,
			currentEntityData: nil,
			want:              false,
		},
		"should return false when deployment ID is missing": {
			currentDeployID:   "",
			currentEntityData: createSensorEntityData("current123", "10.2.2.2"),
			want:              false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockHM := &mockHeritageManager{
				data: []*heritage.SensorMetadata{
					{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
				},
				isEnabled: true,
			}
			store := NewStore(0, mockHM, false)
			store.RememberCurrentSensorMetadata(tt.currentDeployID, tt.currentEntityData)
			got := store.applyHeritageData(mockHM)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestStore_ApplyHeritageDataOnce_Concurrent is a regression test for a "concurrent map writes"
// crash: with the PubSub concurrent lane, deployment events are processed on separate goroutines,
// so ApplyDataFromHeritageOnce could be entered concurrently. Both callers passed the "once" guard
// and mutated the shared currentSensorEntityData maps at the same time. Run with -race.
func TestStore_ApplyHeritageDataOnce_Concurrent(t *testing.T) {
	mockHM := &mockHeritageManager{
		data: []*heritage.SensorMetadata{
			{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
			{ContainerID: "past456", PodIP: "10.1.1.2", SensorStart: time.Now().Add(-2 * time.Hour)},
		},
		isEnabled: true,
	}
	store := NewStore(0, mockHM, false)

	currentData := createSensorEntityData("current123", "10.2.2.2")
	// Endpoints are required to exercise EntityData.AddEndpoint, the site of the original crash.
	currentData.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP), EndpointTargetInfo{ContainerPort: 8443})
	currentData.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP), EndpointTargetInfo{ContainerPort: 9090})
	store.RememberCurrentSensorMetadata("sensor-deploy-1", currentData)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			store.ApplyDataFromHeritageOnce()
		})
	}
	wg.Wait()

	assert.True(t, store.heritageApplied.IsDone())
}

// TestStore_ApplyHeritage_ConcurrentWithApply is a regression test for a data race between the
// heritage apply and a concurrent Apply() call on the same EntityData. onDeploymentCreateOrUpdate
// stores the deployment's EntityData via RememberCurrentSensorMetadata and then passes that same
// object to Apply(); with the PubSub concurrent lane these run on separate goroutines. Serializing
// the heritage apply is not enough: it must not mutate the shared object in place while Apply reads
// it. Run with -race.
func TestStore_ApplyHeritage_ConcurrentWithApply(t *testing.T) {
	mockHM := &mockHeritageManager{
		data: []*heritage.SensorMetadata{
			{ContainerID: "past123", PodIP: "10.1.1.1", SensorStart: time.Now().Add(-time.Hour)},
		},
		isEnabled: true,
	}
	store := NewStore(0, mockHM, false)

	// The same object is remembered as current sensor metadata and passed to Apply below.
	shared := createSensorEntityData("current123", "10.2.2.2")
	shared.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP), EndpointTargetInfo{ContainerPort: 8443})
	shared.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP), EndpointTargetInfo{ContainerPort: 9090})
	store.RememberCurrentSensorMetadata("sensor-deploy-1", shared)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		store.ApplyDataFromHeritageOnce()
	}()
	go func() {
		defer wg.Done()
		store.Apply(map[string]*EntityData{"sensor-deploy-1": shared}, false)
	}()
	wg.Wait()

	assert.True(t, store.heritageApplied.IsDone())
}

func TestApplyPastToEntityData(t *testing.T) {
	tests := map[string]struct {
		currentData    *EntityData
		pastSensor     *heritage.SensorMetadata
		expectedResult bool

		expectedIPs   []net.IPAddress
		unexpectedIPs []net.IPAddress

		expectedEndpoints []net.NumericEndpoint

		expectedContainerIDs []string
	}{
		"should add new heritage data and return true": {
			currentData:          createSensorEntityData("current123", "10.2.2.2"),
			pastSensor:           &heritage.SensorMetadata{ContainerID: "past456", PodIP: "10.1.1.1"},
			expectedResult:       true,
			expectedIPs:          []net.IPAddress{net.ParseIP("10.2.2.2"), net.ParseIP("10.1.1.1")},
			unexpectedIPs:        []net.IPAddress{},
			expectedContainerIDs: []string{"current123", "past456"},
		},
		"should skip existing container and return false": {
			currentData:          createSensorEntityData("duplicate123", "10.2.2.2"),
			pastSensor:           &heritage.SensorMetadata{ContainerID: "duplicate123", PodIP: "10.1.1.1"}, // Same container ID
			expectedResult:       false,
			expectedIPs:          []net.IPAddress{net.ParseIP("10.2.2.2")},
			unexpectedIPs:        []net.IPAddress{net.ParseIP("10.1.1.1")},
			expectedContainerIDs: []string{"duplicate123"},
		},
		"should generate heritage endpoints for past IP": {
			currentData: func() *EntityData {
				data := createSensorEntityData("current123", "10.2.2.2")
				// Add some endpoints to current data
				data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP), EndpointTargetInfo{ContainerPort: 8443})
				data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP), EndpointTargetInfo{ContainerPort: 9090})
				return data
			}(),
			pastSensor:     &heritage.SensorMetadata{ContainerID: "past456", PodIP: "10.1.1.1"},
			expectedResult: true,
			expectedIPs:    []net.IPAddress{net.ParseIP("10.2.2.2"), net.ParseIP("10.1.1.1")},
			unexpectedIPs:  []net.IPAddress{},
			expectedEndpoints: []net.NumericEndpoint{
				net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP),
				net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP),
				net.MakeNumericEndpoint(net.ParseIP("10.1.1.1"), 8443, net.TCP),
				net.MakeNumericEndpoint(net.ParseIP("10.1.1.1"), 9090, net.TCP),
			},
			expectedContainerIDs: []string{"current123", "past456"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := applyPastToEntityData(tt.currentData, maps.Clone(tt.currentData.endpoints), tt.pastSensor)
			assert.Equal(t, tt.expectedResult, result)
			containerIDs := tt.currentData.GetContainerIDs("sensor")
			podIPs := tt.currentData.GetValidIPs()
			gotEndpoints := slices.Collect(maps.Keys(tt.currentData.endpoints))

			// Sort before asserting with ElementsMatch
			slices.SortFunc(podIPs, net.IPAddressCompare)
			slices.SortFunc(tt.expectedIPs, net.IPAddressCompare)

			slices.Sort(containerIDs)
			slices.Sort(tt.expectedContainerIDs)
			slices.SortFunc(tt.expectedEndpoints, net.NumericEndpointCompare)
			slices.SortFunc(gotEndpoints, net.NumericEndpointCompare)

			assert.ElementsMatch(t, tt.expectedIPs, podIPs, "IP should be added")
			assert.ElementsMatch(t, tt.expectedEndpoints, gotEndpoints, "Endpoints should be added")
			for _, ip := range tt.unexpectedIPs {
				assert.NotContains(t, podIPs, ip, "IP should not be added")
			}
			assert.ElementsMatch(t, tt.expectedContainerIDs, containerIDs, "Container IDs should be added")
		})
	}
}

// TestApplyPastToEntityData_NoDuplicateEndpoints is a regression test for endpoints being
// duplicated because applyPastToEntityData added entries to data.endpoints while ranging over it.
// Adding keys to a map during iteration is nondeterministic in Go (a new key "may be produced or
// may be skipped"); with enough endpoints the buggy version reliably re-visited freshly added
// past-IP endpoints and appended their info a second time.
func TestApplyPastToEntityData_NoDuplicateEndpoints(t *testing.T) {
	const nPorts = 64
	data := createSensorEntityData("current123", "10.2.2.2")
	for i := range nPorts {
		port := uint16(1000 + i)
		data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), port, net.TCP), EndpointTargetInfo{ContainerPort: port})
	}

	added := applyPastToEntityData(data, maps.Clone(data.endpoints), &heritage.SensorMetadata{ContainerID: "past456", PodIP: "10.1.1.1"})
	assert.True(t, added)

	// Every endpoint (current and past) must carry exactly one info; a longer slice means the map
	// was mutated while being ranged over and an entry was processed twice.
	assert.Len(t, data.endpoints, 2*nPorts, "expected current + past endpoints and nothing else")
	for ep, infos := range data.endpoints {
		assert.Lenf(t, infos, 1, "endpoint %v has duplicate infos: %v", ep, infos)
	}
}

// TestApplyPastToEntityData_MultiplePastNoCompounding is a regression test for endpoints compounding
// across past entries: applyHeritageData reuses a single EntityData for all past sensors, so deriving
// past endpoints from data.endpoints (which grows each entry) rather than from an immutable snapshot
// caused the per-endpoint info slice to double with every additional past sensor (1, 2, 4, 8, ...).
func TestApplyPastToEntityData_MultiplePastNoCompounding(t *testing.T) {
	data := createSensorEntityData("current123", "10.2.2.2")
	data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP), EndpointTargetInfo{ContainerPort: 8443})
	data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 9090, net.TCP), EndpointTargetInfo{ContainerPort: 9090})

	// Mirror applyHeritageData: snapshot once, then apply every past entry against that snapshot.
	srcEndpoints := maps.Clone(data.endpoints)
	past := []*heritage.SensorMetadata{
		{ContainerID: "past1", PodIP: "10.1.1.1"},
		{ContainerID: "past2", PodIP: "10.1.1.2"},
		{ContainerID: "past3", PodIP: "10.1.1.3"},
		{ContainerID: "past4", PodIP: "10.1.1.4"},
	}
	for _, entry := range past {
		assert.True(t, applyPastToEntityData(data, srcEndpoints, entry))
	}

	// 2 ports for the current IP plus 2 ports per past IP, each with exactly one info.
	assert.Len(t, data.endpoints, 2*(1+len(past)), "unexpected number of endpoints")
	for ep, infos := range data.endpoints {
		assert.Lenf(t, infos, 1, "endpoint %v has duplicate infos: %v", ep, infos)
	}
}

// TestApplyPastToEntityData_PastIPEqualsCurrent is a regression test for a past pod IP that coincides
// with the current Sensor's endpoint IP: deriving a past endpoint would collide with an existing key
// and append a duplicate info to it. Applying the same past entry must leave one info per endpoint.
func TestApplyPastToEntityData_PastIPEqualsCurrent(t *testing.T) {
	data := createSensorEntityData("current123", "10.2.2.2")
	data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.2.2.2"), 8443, net.TCP), EndpointTargetInfo{ContainerPort: 8443})

	srcEndpoints := maps.Clone(data.endpoints)
	// Past sensor reused the current pod IP but has a different container ID.
	assert.True(t, applyPastToEntityData(data, srcEndpoints, &heritage.SensorMetadata{ContainerID: "past456", PodIP: "10.2.2.2"}))

	assert.Len(t, data.endpoints, 1, "no new endpoint expected for a coinciding IP")
	for ep, infos := range data.endpoints {
		assert.Lenf(t, infos, 1, "endpoint %v has duplicate infos: %v", ep, infos)
	}
}

func TestEntityData_String_SlicesCollectFix(t *testing.T) {
	// Test justification: Validates the slices.Collect fix for proper formatting
	tests := map[string]struct {
		setupData  func() *EntityData
		expectsNil bool
	}{
		"should format output with multiple elements properly": {
			setupData: func() *EntityData {
				data := &EntityData{}
				data.AddIP(net.ParseIP("10.1.1.1"))
				data.AddIP(net.ParseIP("10.2.2.2"))
				data.AddEndpoint(net.MakeNumericEndpoint(net.ParseIP("10.1.1.1"), 8443, net.TCP), EndpointTargetInfo{})
				data.AddContainerID("container123", ContainerMetadata{})
				return data
			},
			expectsNil: false,
		},
		"should return nil string for nil entity data": {
			setupData: func() *EntityData {
				return nil
			},
			expectsNil: true,
		},
		"should format empty entity data without nil string": {
			setupData: func() *EntityData {
				return &EntityData{}
			},
			expectsNil: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data := tt.setupData()
			result := data.String()

			if tt.expectsNil {
				assert.Equal(t, "nil", result)
			} else {
				// Verify string contains expected structure - validates slices.Collect fix
				assert.Contains(t, result, "ips:")
				assert.Contains(t, result, "endpoints:")
				assert.Contains(t, result, "containerIDs:")
				assert.NotEmpty(t, result)
				assert.NotContains(t, result, "0x") // Should not contain memory addresses
			}
		})
	}
}

// Helper functions for test setup

func createSensorEntityData(containerID, podIP string) *EntityData {
	data := &EntityData{}
	data.AddIP(net.ParseIP(podIP))
	data.AddContainerID(containerID, ContainerMetadata{
		ContainerName: "sensor",
		ContainerID:   containerID,
	})
	return data
}
