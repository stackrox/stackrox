package nodeinventorizer

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

type mockSleeper struct {
	receivedDuration time.Duration
	callCount        int
}

func (ms *mockSleeper) mockWaitCallback(d time.Duration) {
	ms.receivedDuration = d
	ms.callCount++
}

type mockScanner struct{}

func (m mockScanner) Scan(nodeName string) (*storage.NodeInventory, error) {
	return &storage.NodeInventory{NodeName: nodeName}, nil
}

type TestComplianceCachingSuite struct {
	suite.Suite
}

func TestComplianceCaching(t *testing.T) {
	suite.Run(t, new(TestComplianceCachingSuite))
}

func (s *TestComplianceCachingSuite) writeWrap(wrap *inventoryWrap, path string) {
	jsonWrap, err := json.Marshal(&wrap)
	s.NoError(err)
	err = os.WriteFile(path, jsonWrap, 0600)
	s.NoError(err)
}

// func (s *TestComplianceCachingSuite) readWrap(path string) *inventoryWrap {
//	cacheContents, err := os.ReadFile(path)
//	s.NoError(err)
//
//	var wrap inventoryWrap
//	err = json.Unmarshal(cacheContents, &wrap)
//	s.NoError(err)
//	return &wrap
//}

func (s *TestComplianceCachingSuite) TestMin() {
	initial := 2 * time.Second
	maxBackoff := 10 * time.Second

	actual := min(initial, maxBackoff)

	s.Equal(initial, actual)
}

func (s *TestComplianceCachingSuite) TestMinMaxBackoff() {
	initial := 100 * time.Hour
	maxBackoff := 10 * time.Second

	actual := min(initial, maxBackoff)

	s.Equal(maxBackoff, actual)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoff() {
	initial := 2 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	cs := *NewCachingScanner(mockScanner{}, "", cache, initial, maxBackoff, func(time.Duration) {})
	expectedBackoff := (2 * time.Second) * 2

	newBackoff := cs.calcNextBackoff(initial)

	s.Equal(expectedBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoffUpperBoundary() {
	initial := 8 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	cs := *NewCachingScanner(mockScanner{}, "", cache, initial, maxBackoff, func(time.Duration) {})

	newBackoff := cs.calcNextBackoff(initial)

	s.Equal(maxBackoff, newBackoff)
}

func (s *TestComplianceCachingSuite) TestReadFaultyCachedInventory() {
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	w := &inventoryWrap{
		CacheValidUntil:      time.Now().Add(2 * time.Minute),
		RetryBackoffDuration: 0,
		CachedInventory:      "{\n  \"nodeId\": \"notvalid\", \"LANGUAGE_CVES_UNAVAILABLE\"\n  ]\n}",
	}
	s.writeWrap(w, inventoryCachePath)

	actualInventory, actualValidity := readCachedInventory(inventoryCachePath)

	// both actual values should be empty
	s.Equal((*storage.NodeInventory)(nil), actualInventory)
	s.Equal(time.Time{}, actualValidity)
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryWithoutResultCache() {
	initial := 8 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := *NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})
	nodeName := "testme"

	actual, e := cs.Scan(nodeName)

	s.NoError(e)
	s.Equal(nodeName, actual.GetNodeName())
}

func (s *TestComplianceCachingSuite) TestTriggerNodeInventoryHonorBackoff() {
	sleeper := mockSleeper{callCount: 0}

	initial := 2 * time.Second // must be lower than BackoffDuration in file
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := *NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, sleeper.mockWaitCallback)

	w := &inventoryWrap{
		CacheValidUntil:      time.Time{},
		RetryBackoffDuration: 4 * time.Second,
		CachedInventory:      "",
	}
	s.writeWrap(w, inventoryCachePath)

	_, err := cs.Scan("testme")

	s.NoError(err)
	s.Equal(4*time.Second, sleeper.receivedDuration)
	s.Equal(1, sleeper.callCount)
}

func (s *TestComplianceCachingSuite) TestScanReturnsCachedInventory() {
	initial := 2 * time.Second // must be lower than BackoffDuration in file
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := *NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})

	w := &inventoryWrap{
		CacheValidUntil:      time.Now().Add(2 * time.Minute),
		RetryBackoffDuration: 0,
		CachedInventory:      "{\n  \"nodeId\": \"00000000-0000-0000-0000-000000000000\",\n  \"nodeName\": \"cachedNode\",\n  \"scanTime\": \"2023-11-11T11:11:11.382478080Z\",\n  \"components\": {\n    \"namespace\": \"unknown\"\n  },\n  \"notes\": [\n    \"LANGUAGE_CVES_UNAVAILABLE\"\n  ]\n}",
	}
	s.writeWrap(w, inventoryCachePath)

	actual, err := cs.Scan("testme")

	s.NoError(err)
	s.Equal(actual.NodeName, "cachedNode")
}
