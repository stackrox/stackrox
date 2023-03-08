package nodeinventorizer

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/jsonutil"
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

func (s *TestComplianceCachingSuite) readWrapToInventory(path string) *storage.NodeInventory {
	cacheContents, err := os.ReadFile(path)
	s.NoError(err)

	var wrap inventoryWrap
	err = json.Unmarshal(cacheContents, &wrap)
	s.NoError(err)

	var testInv storage.NodeInventory
	err = jsonutil.JSONToProto(wrap.CachedInventory, &testInv)
	s.NoError(err)

	return &testInv
}

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
		"same unit - a smaller": {
			a:    2 * time.Second,
			b:    10 * time.Second,
			want: 2 * time.Second,
		},
		"same unit - b smaller": {
			a:    10 * time.Second,
			b:    2 * time.Second,
			want: 2 * time.Second,
		},
		"different unit - a smaller": {
			a:    2 * time.Second,
			b:    7 * time.Hour,
			want: 2 * time.Second,
		},
		"different unit - b smaller": {
			a:    2 * time.Second,
			b:    100 * time.Millisecond,
			want: 100 * time.Millisecond,
		},
	}
	for k, v := range cases {
		s.Run(k, func() {
			actual := min(v.a, v.b)
			s.Equal(v.want, actual)
		})
	}
}

func (s *TestComplianceCachingSuite) TestCalcNextBackoffTable() {
	cases := map[string]struct {
		initial         time.Duration
		maxBackoff      time.Duration
		expectedBackoff time.Duration
	}{
		"next backoff should not hit the limit of 30s": {
			initial:         10 * time.Second,
			maxBackoff:      30 * time.Second,
			expectedBackoff: 15 * time.Second,
		},
		"next backoff should not be higher than the limit of 10s": {
			initial:         8 * time.Second,
			maxBackoff:      10 * time.Second,
			expectedBackoff: 10 * time.Second,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			cs := NewCachingScanner(mockScanner{}, "", c.initial, c.initial, c.maxBackoff, func(time.Duration) {})
			s.Equal(c.expectedBackoff, cs.calcNextBackoff(c.initial))
		})
	}
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

func (s *TestComplianceCachingSuite) TestScanWithoutExistingCacheWritesCache() {
	initial := 8 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := *NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})
	nodeName := "testme"

	actual, err := cs.Scan(nodeName)
	ci := s.readWrapToInventory(inventoryCachePath)
	s.NoError(err)

	s.Equal(nodeName, actual.GetNodeName()) // Check that the directly returned result is correct
	s.Equal(nodeName, ci.GetNodeName())     // Check that the scan is written to cache
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
