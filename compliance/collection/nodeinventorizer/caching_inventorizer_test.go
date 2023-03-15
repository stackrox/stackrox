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

func (s *TestComplianceCachingSuite) readWrap(path string) *inventoryWrap {
	cacheContents, err := os.ReadFile(path)
	s.NoError(err)

	var wrap inventoryWrap
	err = json.Unmarshal(cacheContents, &wrap)
	s.NoError(err)

	return &wrap
}

func (s *TestComplianceCachingSuite) wrapToInventory(wrap *inventoryWrap) *storage.NodeInventory {
	var testInv storage.NodeInventory
	err := jsonutil.JSONToProto(wrap.CachedInventory, &testInv)
	s.NoError(err)

	return &testInv
}

func (s *TestComplianceCachingSuite) inventoryToString(inventory *storage.NodeInventory) string {
	strInv, err := jsonutil.ProtoToJSON(inventory, jsonutil.OptCompact)
	s.NoError(err)

	return strInv
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

func (s *TestComplianceCachingSuite) TestCalcNextBackoff() {
	cases := map[string]struct {
		initial         time.Duration
		cacheDuration   time.Duration
		maxBackoff      time.Duration
		expectedBackoff time.Duration
	}{
		"next backoff should not hit the limit of 30s": {
			initial:         10 * time.Second,
			cacheDuration:   20 * time.Second,
			maxBackoff:      30 * time.Second,
			expectedBackoff: 15 * time.Second,
		},
		"next backoff should not be higher than the limit of 10s": {
			initial:         8 * time.Second,
			cacheDuration:   20 * time.Second,
			maxBackoff:      10 * time.Second,
			expectedBackoff: 10 * time.Second,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			cs := NewCachingScanner(mockScanner{}, "", c.initial, c.cacheDuration, c.maxBackoff, func(time.Duration) {})
			s.Equal(c.expectedBackoff, cs.calcNextBackoff(c.initial))
		})
	}
}

// Inventory part of readCacheState
func (s *TestComplianceCachingSuite) TestReadCacheStateInventory() {
	cases := map[string]struct {
		savedInventory    *storage.NodeInventory
		validUntil        time.Time
		expectedInventory *storage.NodeInventory
		expectedBackoff   time.Duration
	}{
		"cached inventory should be returned on success": {
			savedInventory:    &storage.NodeInventory{NodeName: "testnode"},
			validUntil:        time.Now().Add(2 * time.Minute),
			expectedInventory: &storage.NodeInventory{NodeName: "testnode"},
			expectedBackoff:   0,
		},
		"no inventory and no backoff returned when too old": {
			savedInventory:    &storage.NodeInventory{NodeName: "testnode"},
			validUntil:        time.Time{},
			expectedInventory: nil,
			expectedBackoff:   0,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			path := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
			s.writeWrap(&inventoryWrap{
				CacheValidUntil:      c.validUntil,
				RetryBackoffDuration: "0s",
				CachedInventory:      s.inventoryToString(c.savedInventory),
			}, path)
			cs := NewCachingScanner(mockScanner{}, path, 3*time.Second, 3*time.Second, 3*time.Second, func(time.Duration) {})

			actual := cs.readCacheState(path)

			s.Equal(c.expectedInventory, actual.inventory)
			s.Equal(c.expectedBackoff, actual.backoff)
		})
	}
}

func (s *TestComplianceCachingSuite) TestReadCacheStateFaultyCachedInventoryReturnsMaxBackoff() {
	path := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	maxBackoff := 42 * time.Second
	w := &inventoryWrap{
		CacheValidUntil:      time.Now().Add(2 * time.Minute),
		RetryBackoffDuration: "0s",
		CachedInventory:      "{\n  \"nodeId\": \"notvalid\", \"LANGUAGE_CVES_UNAVAILABLE\"\n  ]\n}",
	}
	s.writeWrap(w, path)
	cs := NewCachingScanner(mockScanner{}, path, 3*time.Second, 3*time.Second, maxBackoff, func(time.Duration) {})

	actual := cs.readCacheState(path)

	s.Nil(actual.inventory)
	s.Equal(maxBackoff, actual.backoff)
}

// Backoff part of readCacheState
func (s *TestComplianceCachingSuite) TestReadCacheStateBackoff() {
	maxBackoff := 42 * time.Second
	cases := map[string]struct {
		savedBackoff    string
		expectedBackoff time.Duration
	}{
		"read backoff should return saved duration on success": {
			savedBackoff:    "5s",
			expectedBackoff: 5 * time.Second,
		},
		"read backoff should return 0 if no wrap exists": {
			savedBackoff:    "",
			expectedBackoff: 0,
		},
		"read backoff should return errorBackoff on duration parse error": {
			savedBackoff:    "thisIsNotADuration",
			expectedBackoff: maxBackoff,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			path := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
			if c.savedBackoff != "" {
				s.writeWrap(&inventoryWrap{
					CacheValidUntil:      time.Time{},
					RetryBackoffDuration: c.savedBackoff,
					CachedInventory:      "",
				}, path)
			}
			cs := NewCachingScanner(mockScanner{}, path, 3*time.Second, 3*time.Second, maxBackoff, func(time.Duration) {})

			actual := cs.readCacheState(path)

			s.Equal(c.expectedBackoff, actual.backoff)
		})
	}
}

func (s *TestComplianceCachingSuite) TestReadCacheStateMaxBackoffOnFaultyWrap() {
	path := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	maxBackoff := 42 * time.Second
	brokenWrap := "{\"UnknownKey\":Value}"
	err := os.WriteFile(path, []byte(brokenWrap), 0600)
	s.NoError(err)
	cs := NewCachingScanner(mockScanner{}, path, 3*time.Second, 3*time.Second, maxBackoff, func(time.Duration) {})

	actual := cs.readCacheState(path)

	s.Equal(actual.backoff, maxBackoff)
	s.Nil(actual.inventory)
}

func (s *TestComplianceCachingSuite) TestReadInventoryWrapFaultyUnmarshal() {
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	brokenWrap := "{\"CachedInventory\":42}"
	err := os.WriteFile(inventoryCachePath, []byte(brokenWrap), 0600)
	s.NoError(err)

	actual, err := readInventoryWrap(inventoryCachePath)

	s.Nil(actual)
	s.Error(err)
}

func (s *TestComplianceCachingSuite) TestReadInventoryWrapDoesntExist() {
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())

	actual, err := readInventoryWrap(inventoryCachePath)

	s.Nil(actual)
	s.NoError(err)
}

func (s *TestComplianceCachingSuite) TestScanWithoutExistingCacheWritesCache() {
	initial := 8 * time.Second
	cache := 2 * time.Minute
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})
	nodeName := "testme"

	// Check the directly returned result is correct
	actual, err := cs.Scan(nodeName)
	s.NoError(err)
	s.Equal(nodeName, actual.GetNodeName())

	// Check the cache that is written by cs.Scan is correct
	w := s.readWrap(inventoryCachePath)
	ci := s.wrapToInventory(w)

	s.Equal(nodeName, ci.GetNodeName(), "The result should be written to cache correctly.")
	s.Greater(w.CacheValidUntil.Unix(), time.Now().Unix(), "The cache should be valid to some time in the future.")
}

func (s *TestComplianceCachingSuite) TestScanWriteBackoffOnCacheFail() {
	sleeper := mockSleeper{callCount: 0}

	initial := 2 * time.Second // must be lower than BackoffDuration in file
	cache := 5 * time.Second
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, sleeper.mockWaitCallback)
	nodeName := "testme"

	brokenWrap := "{\"UnknownKey\":Value}"
	err := os.WriteFile(inventoryCachePath, []byte(brokenWrap), 0600)
	s.NoError(err)

	ci, err := cs.Scan(nodeName)

	s.NoError(err)
	s.Equal(maxBackoff, sleeper.receivedDuration, "Scan should have waited for maxBackoff")
	s.Equal(1, sleeper.callCount)
	s.Equal(nodeName, ci.GetNodeName())
}

func (s *TestComplianceCachingSuite) TestScanHonorBackoff() {
	sleeper := mockSleeper{callCount: 0}

	initial := 2 * time.Second // must be lower than BackoffDuration in file
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, sleeper.mockWaitCallback)
	nodeName := "testme"

	w := &inventoryWrap{
		CacheValidUntil:      time.Time{},
		RetryBackoffDuration: "4s",
		CachedInventory:      "",
	}
	s.writeWrap(w, inventoryCachePath)

	ci, err := cs.Scan(nodeName)

	s.NoError(err)
	s.Equal(4*time.Second, sleeper.receivedDuration, "Scan should have waited for 4 seconds.")
	s.Equal(1, sleeper.callCount)
	s.Equal(nodeName, ci.GetNodeName())
}

func (s *TestComplianceCachingSuite) TestScanReturnsCachedInventory() {
	initial := 2 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})
	validUntil := time.Now().Add(2 * time.Minute)

	w := &inventoryWrap{
		CacheValidUntil:      validUntil,
		RetryBackoffDuration: "0s",
		CachedInventory:      "{\n  \"nodeId\": \"00000000-0000-0000-0000-000000000000\",\n  \"nodeName\": \"cachedNode\",\n  \"scanTime\": \"2023-11-11T11:11:11.382478080Z\",\n  \"components\": {\n    \"namespace\": \"unknown\"\n  },\n  \"notes\": [\n    \"LANGUAGE_CVES_UNAVAILABLE\"\n  ]\n}",
	}
	s.writeWrap(w, inventoryCachePath)

	// Check directly returned results
	actual, err := cs.Scan("testme")
	s.NoError(err)
	s.Equal(actual.NodeName, "cachedNode")

	// Check cached results
	wrap := s.readWrap(inventoryCachePath)
	s.WithinDuration(validUntil, wrap.CacheValidUntil, 0, "Cache validity should not change when accessing a cached scan.")
}

func (s *TestComplianceCachingSuite) TestScanRunsNewInventory() {
	initial := 2 * time.Second
	cache := 10 * time.Second
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})
	validUntil := time.Now().Add(-1 * time.Minute)

	w := &inventoryWrap{
		CacheValidUntil:      validUntil,
		RetryBackoffDuration: "0s",
		CachedInventory:      "{\n  \"nodeId\": \"00000000-0000-0000-0000-000000000000\",\n  \"nodeName\": \"cachedNode\",\n  \"scanTime\": \"2023-11-11T11:11:11.382478080Z\",\n  \"components\": {\n    \"namespace\": \"unknown\"\n  },\n  \"notes\": [\n    \"LANGUAGE_CVES_UNAVAILABLE\"\n  ]\n}",
	}
	s.writeWrap(w, inventoryCachePath)

	// Check directly returned results
	actual, err := cs.Scan("testme")
	s.NoError(err)
	s.Equal(actual.NodeName, "testme")

	// Check cached results
	wrap := s.readWrap(inventoryCachePath)
	ci := s.wrapToInventory(wrap)
	s.Equal("testme", ci.GetNodeName(), "Cache should have been updated with new results.")
	s.Greater(wrap.CacheValidUntil.Unix(), time.Now().Unix(), "The cache should be valid to some time in the future.")
}

func (s *TestComplianceCachingSuite) TestScanFailsOnBackoffWrite() {
	initial := 2 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := s.T().TempDir() // Write will fail as the cache expects a full path to a filename
	cs := NewCachingScanner(mockScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})

	actual, err := cs.Scan("testme")

	s.Nil(actual)
	s.Error(err)
}

type erroringScanner struct{}

func (s erroringScanner) Scan(nodeName string) (*storage.NodeInventory, error) {
	return nil, errors.New("Cached Inventorizer Test Error")
}

func (s *TestComplianceCachingSuite) TestScanFailsOnScannerError() {
	initial := 2 * time.Second
	cache := initial
	maxBackoff := 10 * time.Second
	inventoryCachePath := fmt.Sprintf("%s/inventory-cache", s.T().TempDir())
	cs := NewCachingScanner(erroringScanner{}, inventoryCachePath, cache, initial, maxBackoff, func(time.Duration) {})

	inv, err := cs.Scan("testme")
	s.Error(err)
	s.Nil(inv)

	wrap := s.readWrap(inventoryCachePath)
	s.NotZero(wrap.RetryBackoffDuration, "Backoff duration must not be 0 when scan fails.")
}
