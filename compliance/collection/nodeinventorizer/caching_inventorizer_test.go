package nodeinventorizer

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
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
	cases := map[string]struct {
		a    time.Duration
		b    time.Duration
		want time.Duration
	}{
		"a smaller": {
			a:    time.Second,
			b:    time.Hour,
			want: time.Second,
		},
		"b smaller": {
			a:    time.Hour,
			b:    time.Second,
			want: time.Second,
		},
		"a zero": {
			a:    0 * time.Second,
			b:    time.Second,
			want: 0,
		},
		"concrete values - a smaller": {
			a:    2 * time.Second,
			b:    10 * time.Second,
			want: 2 * time.Second,
		},
		"concrete values - b smaller": {
			a:    10 * time.Second,
			b:    2 * time.Second,
			want: 2 * time.Second,
		},
	}
	for k, v := range cases {
		s.Run(k, func() {
			actual := min(v.a, v.b)
			s.Equal(v.want, actual)
		})
	}
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoff() {
	initial := 10 * time.Second
	cache := initial
	maxBackoff := 30 * time.Second
	cs := *NewCachingScanner(mockScanner{}, "", cache, initial, maxBackoff, func(time.Duration) {})
	expectedBackoff := 15 * time.Second // expected is backoffMultiplier * initial

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

func (s *TestComplianceCachingSuite) TestReadInventoryWrapFaultyUnmarshal() {
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	brokenWrap := "{\"CachedInventory\":42}"
	err := os.WriteFile(inventoryCachePath, []byte(brokenWrap), 0600)
	s.NoError(err)

	actual := readInventoryWrap(inventoryCachePath)

	s.Nil(actual) // We expect nil, but no panic even when the unmarshal fails
}

func (s *TestComplianceCachingSuite) TestScanReadFaultyCachedInventory() {
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	w := &inventoryWrap{
		CacheValidUntil:      time.Now().Add(2 * time.Minute),
		RetryBackoffDuration: 0,
		CachedInventory:      "{\n  \"nodeId\": \"notvalid\", \"LANGUAGE_CVES_UNAVAILABLE\"\n  ]\n}",
	}
	s.writeWrap(w, inventoryCachePath)

	actualInventory, actualValidity := readCachedInventory(inventoryCachePath)

	// both actual values should be empty
	s.Nil(actualInventory)
	s.Equal(time.Time{}, actualValidity)
}

func (s *TestComplianceCachingSuite) TestScanWithoutResultCache() {
	initial := 8 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := *NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})
	nodeName := "testme"

	actual, err := cs.Scan(nodeName)

	s.NoError(err)
	s.Equal(nodeName, actual.GetNodeName())
}

func (s *TestComplianceCachingSuite) TestScanHonorBackoff() {
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
	initial := 2 * time.Second
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

func (s *TestComplianceCachingSuite) TestScanFailsOnBackoffWrite() {
	initial := 2 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := s.T().TempDir() // Write will fail as the cache expects a full path to a filename
	cs := *NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})

	_, err := cs.Scan("testme")

	s.Error(err)
}

type scannerErr struct{}

func (s scannerErr) Scan(nodeName string) (*storage.NodeInventory, error) {
	return nil, errors.New("Cached Inventorizer Test Error")
}

func (s *TestComplianceCachingSuite) TestScanFailsOnScannerError() {
	initial := 2 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := *NewCachingScanner(scannerErr{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})

	_, err := cs.Scan("testme")

	s.Error(err)
}
